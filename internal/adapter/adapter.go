// Package adapter wraps loom/poly for construct-tcp-examples (CPU-MC training).
package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/openfluke/loom/poly"
)

type DenseLayer struct {
	InputSize  int
	OutputSize int
	Activation poly.ActivationType
}

const (
	ActivationScaledReLU = poly.ActivationReLU
	ActivationTanh       = poly.ActivationTanh
	ActivationLinear     = poly.ActivationLinear
)

type TrainingConfig struct {
	Epochs       int
	LearningRate float32
	UseGPU       bool
	Verbose      bool
	LossType     string
	BatchSize    int
}

type TrainingResult struct {
	FinalLoss   float64
	TotalTime   time.Duration
	LossHistory []float64
}

type Network struct {
	Net       *poly.VolumetricNetwork
	BatchSize int
	GPU       bool

	enabled       []bool
	cpuReady      bool
	lastPre       []*poly.Tensor[float32]
	lastIn        []*poly.Tensor[float32]
	lastOut       *poly.Tensor[float32]
	lastGradW     []*poly.Tensor[float32]
	lastTrainLoss float64
}

func NewNetwork(_ int, _, layersPerCell, _ int) *Network {
	if layersPerCell <= 0 {
		layersPerCell = 1
	}
	return &Network{
		Net:       poly.NewVolumetricNetwork(1, 1, 1, layersPerCell),
		BatchSize: 1,
		enabled:   make([]bool, layersPerCell),
		lastGradW: make([]*poly.Tensor[float32], layersPerCell),
	}
}

func InitDenseLayer(input, output int, activation poly.ActivationType) DenseLayer {
	return DenseLayer{InputSize: input, OutputSize: output, Activation: activation}
}

func (n *Network) SetLayer(_, layerIdx, _ int, layer DenseLayer) {
	if layerIdx < 0 || layerIdx >= n.Net.LayersPerCell {
		return
	}
	l := n.Net.GetLayer(0, 0, 0, layerIdx)
	l.Type = poly.LayerDense
	l.InputHeight = layer.InputSize
	l.OutputHeight = layer.OutputSize
	l.Activation = layer.Activation
	l.DType = poly.DTypeFloat32
	wCount := layer.InputSize * layer.OutputSize
	l.WeightStore = poly.NewWeightStore(wCount)
	l.WeightStore.HeRandomize(time.Now().UnixNano()+int64(layerIdx), layer.InputSize)
	n.enabled[layerIdx] = true
}

func (n *Network) InitializeWeights() {
	for i := 0; i < n.Net.LayersPerCell; i++ {
		if !n.enabled[i] {
			continue
		}
		l := n.Net.GetLayer(0, 0, 0, i)
		if l.WeightStore == nil {
			wCount := l.InputHeight * l.OutputHeight
			l.WeightStore = poly.NewWeightStore(wCount)
		}
		l.WeightStore.HeRandomize(time.Now().UnixNano()+int64(i), maxInt(1, l.InputHeight))
	}
}

func (n *Network) ensureCPUMode() error {
	if n.cpuReady {
		return nil
	}
	if err := poly.ConfigureNetworkForMode(n.Net, poly.TrainingModeCPUMC); err != nil {
		return err
	}
	n.Net.EnsureTrainingWeights()
	n.cpuReady = true
	return nil
}

func (n *Network) Forward(input []float32) ([]float32, error) {
	first := n.firstLayer()
	if first == nil || first.InputHeight <= 0 {
		return nil, fmt.Errorf("network not configured")
	}
	if len(input) == 0 || len(input)%first.InputHeight != 0 {
		return nil, fmt.Errorf("input shape mismatch")
	}
	batch := len(input) / first.InputHeight

	cur := poly.NewTensor[float32](batch, first.InputHeight)
	copy(cur.Data, input)
	n.lastPre = make([]*poly.Tensor[float32], n.Net.LayersPerCell)
	n.lastIn = make([]*poly.Tensor[float32], n.Net.LayersPerCell)

	for i := 0; i < n.Net.LayersPerCell; i++ {
		if !n.enabled[i] {
			continue
		}
		layer := n.Net.GetLayer(0, 0, 0, i)
		n.lastIn[i] = cur
		pre, post := poly.DispatchLayer(layer, cur, nil)
		n.lastPre[i] = pre
		cur = post
	}

	n.lastOut = cur
	out := make([]float32, len(cur.Data))
	copy(out, cur.Data)
	return out, nil
}

func (n *Network) Backward(grad []float32) error {
	if n.lastOut == nil || len(grad) != len(n.lastOut.Data) {
		return fmt.Errorf("backward called without matching forward")
	}
	g := poly.NewTensor[float32](n.lastOut.Shape...)
	copy(g.Data, grad)
	for i := n.Net.LayersPerCell - 1; i >= 0; i-- {
		if !n.enabled[i] {
			continue
		}
		layer := n.Net.GetLayer(0, 0, 0, i)
		gIn, gW := poly.DispatchLayerBackward(layer, g, n.lastIn[i], nil, n.lastPre[i])
		n.lastGradW[i] = gW
		g = gIn
	}
	return nil
}

func (n *Network) ApplyGradients(lr float32) {
	for i := 0; i < n.Net.LayersPerCell; i++ {
		if !n.enabled[i] || n.lastGradW[i] == nil {
			continue
		}
		poly.ApplyRecursiveGradients(n.Net.GetLayer(0, 0, 0, i), n.lastGradW[i], lr, 0)
	}
}

func (n *Network) SaveModel(filename, _ string) error {
	b, err := poly.SerializeNetwork(n.Net)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, b, 0o644)
}

func LoadModel(filename, _ string) (*Network, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	net, err := poly.DeserializeNetwork(b)
	if err != nil {
		return nil, err
	}
	n := &Network{
		Net:       net,
		BatchSize: 1,
		enabled:   make([]bool, net.LayersPerCell),
		lastGradW: make([]*poly.Tensor[float32], net.LayersPerCell),
	}
	for i := 0; i < net.LayersPerCell; i++ {
		n.enabled[i] = !net.GetLayer(0, 0, 0, i).IsDisabled
	}
	return n, nil
}

func DefaultTrainingConfig() *TrainingConfig {
	return &TrainingConfig{Epochs: 10, LearningRate: 0.01, LossType: "mse", BatchSize: 64}
}

func (n *Network) lastOutputSize() int {
	for i := n.Net.LayersPerCell - 1; i >= 0; i-- {
		if n.enabled[i] {
			return n.Net.GetLayer(0, 0, 0, i).OutputHeight
		}
	}
	return 0
}

func (n *Network) buildBatches(inputs, targets [][]float32, batchSize int) ([]poly.TrainingBatch[float32], error) {
	if len(inputs) == 0 || len(inputs) != len(targets) {
		return nil, fmt.Errorf("invalid training data")
	}
	first := n.firstLayer()
	outSize := n.lastOutputSize()
	if first == nil || outSize <= 0 {
		return nil, fmt.Errorf("network not configured")
	}
	if batchSize <= 0 {
		batchSize = 64
	}

	var batches []poly.TrainingBatch[float32]
	for i := 0; i < len(inputs); i += batchSize {
		end := i + batchSize
		if end > len(inputs) {
			end = len(inputs)
		}
		nSamples := end - i
		inT := poly.NewTensor[float32](nSamples, first.InputHeight)
		tgtT := poly.NewTensor[float32](nSamples, outSize)
		for j := 0; j < nSamples; j++ {
			copy(inT.Data[j*first.InputHeight:(j+1)*first.InputHeight], inputs[i+j])
			copy(tgtT.Data[j*outSize:(j+1)*outSize], targets[i+j])
		}
		batches = append(batches, poly.TrainingBatch[float32]{Input: inT, Target: tgtT})
	}
	return batches, nil
}

func (n *Network) TrainFromSamples(inputs, targets [][]float32, cfg *TrainingConfig) (*TrainingResult, error) {
	if cfg == nil {
		cfg = DefaultTrainingConfig()
	}
	if err := n.ensureCPUMode(); err != nil {
		return nil, err
	}
	batches, err := n.buildBatches(inputs, targets, cfg.BatchSize)
	if err != nil {
		return nil, err
	}
	pcfg := &poly.TrainingConfig{
		Epochs:       cfg.Epochs,
		LearningRate: cfg.LearningRate,
		LossType:     "mse",
		Verbose:      cfg.Verbose,
		Mode:         poly.TrainingModeCPUMC,
		GradientClip: 1.0,
	}
	if cfg.LossType != "" {
		pcfg.LossType = cfg.LossType
	}
	res, err := poly.Train(n.Net, batches, pcfg)
	if err != nil {
		return nil, err
	}
	n.lastTrainLoss = res.FinalLoss
	return &TrainingResult{
		FinalLoss:   res.FinalLoss,
		TotalTime:   res.TotalTime,
		LossHistory: res.LossHistory,
	}, nil
}

func (n *Network) TrainOneBatch(states, targets []float32, lr float32) (float64, error) {
	if err := n.ensureCPUMode(); err != nil {
		return 0, err
	}
	first := n.firstLayer()
	outSize := n.lastOutputSize()
	if first == nil || outSize <= 0 {
		return 0, fmt.Errorf("network not configured")
	}
	if len(states) == 0 || len(states)%first.InputHeight != 0 {
		return 0, fmt.Errorf("state shape mismatch")
	}
	batch := len(states) / first.InputHeight
	if len(targets) != batch*outSize {
		return 0, fmt.Errorf("target shape mismatch")
	}

	inT := poly.NewTensor[float32](batch, first.InputHeight)
	tgtT := poly.NewTensor[float32](batch, outSize)
	copy(inT.Data, states)
	copy(tgtT.Data, targets)

	batches := []poly.TrainingBatch[float32]{{Input: inT, Target: tgtT}}
	pcfg := &poly.TrainingConfig{
		Epochs:       1,
		LearningRate: lr,
		LossType:     "mse",
		Verbose:      false,
		Mode:         poly.TrainingModeCPUMC,
		GradientClip: 1.0,
	}
	res, err := poly.Train(n.Net, batches, pcfg)
	if err != nil {
		return 0, err
	}
	n.lastTrainLoss = res.FinalLoss
	return res.FinalLoss, nil
}

func (n *Network) LastTrainLoss() float64 { return n.lastTrainLoss }

func (n *Network) TrainStandard(inputs, targets [][]float32, cfg *TrainingConfig) (*TrainingResult, error) {
	return n.TrainFromSamples(inputs, targets, cfg)
}

func (n *Network) firstLayer() *poly.VolumetricLayer {
	for i := 0; i < n.Net.LayersPerCell; i++ {
		if n.enabled[i] {
			return n.Net.GetLayer(0, 0, 0, i)
		}
	}
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func VerboseEpochJSON(epoch int, loss float64) {
	msg, _ := json.Marshal(map[string]any{"epoch": epoch, "loss": loss})
	fmt.Println(string(msg))
}
