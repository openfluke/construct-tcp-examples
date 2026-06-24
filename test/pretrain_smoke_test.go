package test

import (
	"math/rand"
	"testing"

	"github.com/openfluke/construct-tcp-examples/internal/adapter"
)

func TestPretrainLossDecreases(t *testing.T) {
	rand.Seed(1)
	inputSize, outputSize := 15, 3
	network := adapter.NewNetwork(inputSize, 1, 3, 1)
	network.SetLayer(0, 0, 0, adapter.InitDenseLayer(inputSize, 64, adapter.ActivationScaledReLU))
	network.SetLayer(0, 1, 0, adapter.InitDenseLayer(64, 32, adapter.ActivationScaledReLU))
	network.SetLayer(0, 2, 0, adapter.InitDenseLayer(32, outputSize, adapter.ActivationLinear))
	network.InitializeWeights()

	inputs, targets := adapter.GenerateExpertData(512)
	cfg := adapter.DefaultTrainingConfig()
	cfg.Epochs = 10
	cfg.LearningRate = 0.005
	cfg.BatchSize = 64

	res, err := network.TrainFromSamples(inputs, targets, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.LossHistory) < 2 {
		t.Fatal("expected loss history")
	}
	first, last := res.LossHistory[0], res.LossHistory[len(res.LossHistory)-1]
	if last >= first {
		t.Fatalf("loss did not decrease: first=%.6f last=%.6f", first, last)
	}
}
