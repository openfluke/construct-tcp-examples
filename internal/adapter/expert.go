package adapter

import (
	"math"
	"math/rand"
)

// GenerateExpertData builds normalized swarm-RL pre-training samples (test10).
func GenerateExpertData(numSamples int) ([][]float32, [][]float32) {
	inputs := make([][]float32, numSamples)
	targets := make([][]float32, numSamples)
	planetCenter := vec3{0, 0, 0}
	planetRadius := float32(100)

	for i := 0; i < numSamples; i++ {
		px := (rand.Float32() - 0.5) * 200
		py := rand.Float32()*40 + 80
		pz := (rand.Float32() - 0.5) * 200
		tx := px + (rand.Float32()-0.5)*80
		ty := py + (rand.Float32()-0.5)*20
		tz := pz + (rand.Float32()-0.5)*80

		pos := vec3{px, py, pz}
		rot := vec3{rand.Float32() * 360, rand.Float32() * 360, rand.Float32() * 360}
		vel := vec3{rand.Float32()*2 - 1, rand.Float32()*2 - 1, rand.Float32()*2 - 1}
		angVel := vec3{rand.Float32()*0.2 - 0.1, rand.Float32()*0.2 - 0.1, rand.Float32()*0.2 - 0.1}
		target := vec3{tx, ty, tz}

		inputs[i] = normalizeRLState(pos, rot, vel, angVel, target, planetCenter, planetRadius)
		forward := forwardFromRotation(rot)
		targetDir := vecNorm(vecSub(target, pos))
		torque := expertTorqueAction(forward, targetDir)
		targets[i] = []float32{torque[0], torque[1], torque[2]}
	}
	return inputs, targets
}

type vec3 [3]float32

func normalizeRLState(pos, rot, vel, angVel, target, planetCenter vec3, planetRadius float32) []float32 {
	if planetRadius <= 0 {
		planetRadius = 100
	}
	rel := vecMul(vecSub(pos, planetCenter), 1.0/planetRadius)
	toTarget := vecNorm(vecSub(target, pos))
	return []float32{
		rel[0], rel[1], rel[2],
		rot[0] / 180.0, rot[1] / 180.0, rot[2] / 180.0,
		clampUnit(vel[0] / 8.0), clampUnit(vel[1] / 8.0), clampUnit(vel[2] / 8.0),
		clampUnit(angVel[0]), clampUnit(angVel[1]), clampUnit(angVel[2]),
		toTarget[0], toTarget[1], toTarget[2],
	}
}

func clampUnit(v float32) float32 {
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}

func expertTorqueAction(forward, targetDir vec3) vec3 {
	cross := vecCross(forward, targetDir)
	strength := float32(math.Sqrt(float64(vecDot(cross, cross))))
	if strength < 1e-4 {
		return vec3{0, 0, 0}
	}
	axis := vecMul(cross, 1.0/strength)
	out := vecMul(axis, strength)
	maxC := float32(math.Max(math.Max(math.Abs(float64(out[0])), math.Abs(float64(out[1]))), math.Abs(float64(out[2]))))
	if maxC > 1 {
		out = vecMul(out, 1.0/maxC)
	}
	return out
}

func forwardFromRotation(rot vec3) vec3 {
	radX := float64(rot[0]) * math.Pi / 180.0
	radY := float64(rot[1]) * math.Pi / 180.0
	return vec3{
		float32(-math.Sin(radY)),
		float32(math.Sin(radX) * math.Cos(radY)),
		float32(-math.Cos(radX) * math.Cos(radY)),
	}
}

func vecSub(a, b vec3) vec3     { return vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func vecMul(v vec3, s float32) vec3 { return vec3{v[0] * s, v[1] * s, v[2] * s} }
func vecDot(a, b vec3) float32 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}
func vecLen(v vec3) float32 {
	return float32(math.Sqrt(float64(vecDot(v, v))))
}
func vecNorm(v vec3) vec3 {
	l := vecLen(v)
	if l < 1e-6 {
		return vec3{0, 0, 0}
	}
	return vecMul(v, 1.0/l)
}
func vecCross(a, b vec3) vec3 {
	return vec3{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}
