package main

import "github.com/openfluke/construct-tcp-examples/internal/adapter"

type (
	Network        = adapter.Network
	DenseLayer     = adapter.DenseLayer
	TrainingConfig = adapter.TrainingConfig
	TrainingResult = adapter.TrainingResult
)

const (
	ActivationScaledReLU = adapter.ActivationScaledReLU
	ActivationTanh       = adapter.ActivationTanh
	ActivationLinear     = adapter.ActivationLinear
)

var (
	NewNetwork            = adapter.NewNetwork
	InitDenseLayer        = adapter.InitDenseLayer
	LoadModel             = adapter.LoadModel
	DefaultTrainingConfig = adapter.DefaultTrainingConfig
)

func verboseEpochJSON(epoch int, loss float64) {
	adapter.VerboseEpochJSON(epoch, loss)
}

func generateExpertData(numSamples int) ([][]float32, [][]float32) {
	return adapter.GenerateExpertData(numSamples)
}
