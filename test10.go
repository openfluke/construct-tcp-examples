package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"time"
)

// Agent represents a swarm agent learning to orient towards targets
type Agent struct {
	ID              string
	TargetPos       Vector3
	CurrentPos      Vector3
	PlanetCenter    Vector3
	PlanetRadius    float32
	Rotation        Vector3
	Velocity        Vector3 // Linear velocity
	AngularVelocity Vector3 // Angular velocity
	LastPos         Vector3 // For velocity calculation
	LastReward      float32
	Color           Vector3
	TotalReward     float32
	StepCount       int
}

func (a *Agent) GetForward() Vector3 {
	return GetForwardFromRotation(a.Rotation)
}

// Helper to calculate forward vector from rotation
func GetForwardFromRotation(rot Vector3) Vector3 {
	radX := float64(rot[0]) * math.Pi / 180.0
	radY := float64(rot[1]) * math.Pi / 180.0
	fx := float32(-math.Sin(radY))
	fy := float32(math.Sin(radX) * math.Cos(radY))
	fz := float32(-math.Cos(radX) * math.Cos(radY))
	return Vector3{fx, fy, fz}
}

func (a *Agent) GetState() []float32 {
	return normalizeRLState(a.CurrentPos, a.Rotation, a.Velocity, a.AngularVelocity, a.TargetPos, a.PlanetCenter, a.PlanetRadius)
}

// normalizeRLState scales features to ~[-1, 1] for stable loom CPU training.
func normalizeRLState(pos, rot, vel, angVel, target, planetCenter Vector3, planetRadius float32) []float32 {
	if planetRadius <= 0 {
		planetRadius = 100
	}
	rel := VecMul(VecSub(pos, planetCenter), 1.0/planetRadius)
	toTarget := VecNorm(VecSub(target, pos))
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

// expertTorqueAction returns a [-1,1] torque hint pointing forward toward target.
func expertTorqueAction(forward, targetDir Vector3) Vector3 {
	cross := Cross(forward, targetDir)
	strength := float32(math.Sqrt(float64(VecDot(cross, cross))))
	if strength < 1e-4 {
		return Vector3{0, 0, 0}
	}
	axis := VecMul(cross, 1.0/strength)
	out := VecMul(axis, strength)
	maxC := float32(math.Max(math.Max(math.Abs(float64(out[0])), math.Abs(float64(out[1]))), math.Abs(float64(out[2]))))
	if maxC > 1 {
		out = VecMul(out, 1.0/maxC)
	}
	return out
}

// Experience represents a single (s, a, r, s', done) tuple for replay
type Experience struct {
	State     []float32
	Action    []float32 // Changed from Vector3 to support variable action sizes
	Reward    float32
	NextState []float32
	Done      bool
}

// ExperienceBuffer implements a circular buffer for experience replay
type ExperienceBuffer struct {
	Buffer   []Experience
	Capacity int
	Index    int
	Size     int
}

func NewExperienceBuffer(capacity int) *ExperienceBuffer {
	return &ExperienceBuffer{
		Buffer:   make([]Experience, capacity),
		Capacity: capacity,
		Index:    0,
		Size:     0,
	}
}

func (eb *ExperienceBuffer) Add(exp Experience) {
	eb.Buffer[eb.Index] = exp
	eb.Index = (eb.Index + 1) % eb.Capacity
	if eb.Size < eb.Capacity {
		eb.Size++
	}
}

func (eb *ExperienceBuffer) Sample(batchSize int) []Experience {
	if eb.Size < batchSize {
		batchSize = eb.Size
	}
	batch := make([]Experience, batchSize)
	for i := 0; i < batchSize; i++ {
		idx := rand.Intn(eb.Size)
		batch[i] = eb.Buffer[idx]
	}
	return batch
}

func (eb *ExperienceBuffer) IsFull() bool {
	return eb.Size >= eb.Capacity
}

// TrainingMetrics tracks training progress
type TrainingMetrics struct {
	Episode   int
	AvgReward float32
	Loss      float32
	Epsilon   float32
	Timestep  int
}

func saveMetrics(filename string, metrics []TrainingMetrics) {
	file, err := os.Create(filename)
	if err != nil {
		return
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	w.Write([]string{"episode", "avg_reward", "loss", "epsilon", "timestep"})
	for _, m := range metrics {
		w.Write([]string{
			fmt.Sprintf("%d", m.Episode),
			fmt.Sprintf("%.4f", m.AvgReward),
			fmt.Sprintf("%.6f", m.Loss),
			fmt.Sprintf("%.4f", m.Epsilon),
			fmt.Sprintf("%d", m.Timestep),
		})
	}
}

// Calculate shaped reward for better learning signal (alignment + tangent progress).
func calculateReward(agent *Agent) float32 {
	fwd := agent.GetForward()
	targetDir := VecNorm(VecSub(agent.TargetPos, agent.CurrentPos))
	alignment := VecDot(fwd, targetDir)
	moveDir := tangentToward(agent.CurrentPos, agent.PlanetCenter, agent.TargetPos)
	progress := VecDot(agent.Velocity, moveDir)
	reward := alignment + progress*0.3
	if alignment > 0.85 {
		reward += 0.3
	}
	return reward
}

// tangentToward returns unit direction on the planet tangent plane pointing at target.
func tangentToward(pos, planetCenter, target Vector3) Vector3 {
	up := VecNorm(VecSub(pos, planetCenter))
	toTarget := VecSub(target, pos)
	tangent := VecSub(toTarget, VecMul(up, VecDot(toTarget, up)))
	if VecDot(tangent, tangent) < 1e-6 {
		return Vector3{0, 0, 0}
	}
	return VecNorm(tangent)
}

func preTrainNetwork(network *Network) {
	fmt.Println("🛰️  Starting Pre-Training on Expert Demonstrations (loom CPU-MC)...")

	numSamples := 5000
	inputs, targets := generateExpertData(numSamples)

	config := DefaultTrainingConfig()
	config.Epochs = 20
	config.LearningRate = 0.005
	config.BatchSize = 128
	config.Verbose = false
	config.LossType = "mse"

	result, err := network.TrainFromSamples(inputs, targets, config)
	if err != nil {
		fmt.Printf("❌ Pre-training failed: %v\n", err)
		return
	}

	for i, loss := range result.LossHistory {
		verboseEpochJSON(i+1, loss)
	}

	fmt.Printf("✅ Pre-Training Complete - Final Loss: %.6f, Time: %v\n",
		result.FinalLoss, result.TotalTime)
}

func RunTest10() {
	fmt.Println("🧠 Starting Test 10: SWARM REINFORCEMENT LEARNING 🧠")

	// 1. Connect to PrimeCraft server
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ Failed to connect to server: %v\n", err)
		return
	}
	defer conn.Close()

	// 2. Discover environment (bubbles)
	fmt.Println("📡 Querying world state for bubbles...")
	writePacket(conn, []byte(`{"type":"query_state"}`))

	buf := make([]byte, 32768)
	n, _ := conn.Read(buf)
	var state StateResponse
	json.Unmarshal(buf[:n], &state)

	if len(state.Bubbles) == 0 {
		fmt.Println("⚠️  No bubbles found. Cannot start RL training.")
		return
	}

	numBubbles := len(state.Bubbles)
	if numBubbles > 10 {
		numBubbles = 10
	}
	agentsPerBubble := 10
	numAgents := numBubbles * agentsPerBubble

	fmt.Printf("✅ Found %d bubbles. Initializing %d agents...\n", numBubbles, numAgents)

	planetCenter := Vector3{state.PlanetCenter[0], state.PlanetCenter[1], state.PlanetCenter[2]}

	// 3. Create neural network with BATCHED processing
	inputSize := 15 // [pos(3), rot(3), vel(3), angvel(3), target(3)]
	outputSize := 3 // [pitch_torque, yaw_torque, roll_torque]

	fmt.Println("🏗️  Building Batched Neural Network...")
	fmt.Printf("   - Input: %d features per agent\n", inputSize)
	fmt.Printf("   - Architecture: Dense(%d → 64 → 32 → %d)\n", inputSize, outputSize)
	fmt.Printf("   - Batch size: %d agents processed together\n", numAgents)

	// Simple feedforward: Input(15) → Dense(64) → Dense(32) → Dense(3)
	network := NewNetwork(inputSize, 1, 3, 1)
	network.BatchSize = numAgents // Process all agents in one batch

	layer0 := InitDenseLayer(inputSize, 64, ActivationScaledReLU)
	layer1 := InitDenseLayer(64, 32, ActivationScaledReLU)
	layer2 := InitDenseLayer(32, outputSize, ActivationLinear)

	network.SetLayer(0, 0, 0, layer0)
	network.SetLayer(0, 1, 0, layer1)
	network.SetLayer(0, 2, 0, layer2)

	fmt.Println("✅ Training backend: loom/poly CPU-MC (ConfigureNetworkForMode + Train)")

	// Try to load latest checkpoint FIRST
	checkpointLoaded := false
	latestCheckpoint := ""
	latestEpisode := 0

	// Search for checkpoint files
	files, err := os.ReadDir(".")
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && len(file.Name()) > 24 && file.Name()[:24] == "swarm_model_checkpoint_" {
				// Extract episode number from filename
				var ep int
				if _, err := fmt.Sscanf(file.Name(), "swarm_model_checkpoint_%d.bin", &ep); err == nil {
					if ep > latestEpisode {
						latestEpisode = ep
						latestCheckpoint = file.Name()
					}
				}
			}
		}
	}

	if latestCheckpoint != "" {
		fmt.Printf("📂 Found checkpoint: %s (Episode %d)\n", latestCheckpoint, latestEpisode)
		loadedNetwork, err := LoadModel(latestCheckpoint, fmt.Sprintf("swarm_ep_%d", latestEpisode))
		if err == nil {
			network = loadedNetwork
			network.GPU = false
			checkpointLoaded = true
			fmt.Println("✅ Checkpoint loaded - resuming training!")
		} else {
			fmt.Printf("⚠️  Failed to load checkpoint: %v\n", err)
			fmt.Println("Initializing fresh network...")
		}
	}

	// Only initialize weights if no checkpoint was loaded
	if !checkpointLoaded {
		network.InitializeWeights()
		fmt.Println("✅ Fresh network — CPU-MC pre-training next")
		// Pre-train on expert demonstrations
		preTrainNetwork(network)
	} else {
		fmt.Println("✅ Resuming from checkpoint (CPU-MC)")
	}

	// Metrics log (online training uses live expert labels, not replay buffer).
	metricsHistory := []TrainingMetrics{}

	// 6. Spawn agents
	agents := make([]*Agent, numAgents)
	spawnOffset := float32(3.0) // above bubble on planet surface

	fmt.Printf("🚀 Spawning %d agents across %d bubbles...\n", numAgents, numBubbles)

	agentIdx := 0
	for i := 0; i < numBubbles; i++ {
		b := state.Bubbles[i]
		bPos := Vector3{b.Pos[0], b.Pos[1], b.Pos[2]}
		up := VecNorm(VecSub(bPos, planetCenter))

		nextIdx := (i + 1) % numBubbles
		targetBubble := state.Bubbles[nextIdx]
		targetPos := Vector3{targetBubble.Pos[0], targetBubble.Pos[1], targetBubble.Pos[2]}

		for j := 0; j < agentsPerBubble; j++ {
			theta := (float64(j) / float64(agentsPerBubble)) * 2.0 * math.Pi
			ringDist := float32(12.0)
			right, _, forward := MakeBasis(up)
			localOffset := Vector3{
				float32(math.Cos(theta)) * ringDist,
				0,
				float32(math.Sin(theta)) * ringDist,
			}
			worldOffset := TransformPoint(Vector3{0, 0, 0}, right, up, forward, localOffset)

			spawnPos := Vector3{
				bPos[0] + up[0]*spawnOffset + worldOffset[0],
				bPos[1] + up[1]*spawnOffset + worldOffset[1],
				bPos[2] + up[2]*spawnOffset + worldOffset[2],
			}

			initialRot := GetEuler(GetBasis(up))
			id := fmt.Sprintf("rl_cube_%d", agentIdx)

			agents[agentIdx] = &Agent{
				ID:              id,
				CurrentPos:      spawnPos,
				LastPos:         spawnPos,
				PlanetCenter:    planetCenter,
				PlanetRadius:    state.PlanetRadius,
				Rotation:        Vector3{rand.Float32() * 360, rand.Float32() * 360, 0},
				TargetPos:       targetPos,
				Velocity:        Vector3{0, 0, 0}, // Start at rest
				AngularVelocity: Vector3{0, 0, 0}, // Start at rest
				Color:           Vector3{0.2, 0.5, 1.0},
			}

			createReq := ConstructRequest{
				Type:        "create_construct",
				ConstructID: id,
				Parts: []Part{
					{
						ID:     "core",
						Type:   "box",
						Size:   Vector3{2, 2, 2},
						Pos:    spawnPos,
						Rot:    initialRot,
						Color:  agents[agentIdx].Color,
						Locked: false,
						Groups: []string{"lasso_target", "rl_agent"},
					},
				},
			}

			data, _ := json.Marshal(createReq)
			writePacket(conn, data)
			agentIdx++
			time.Sleep(10 * time.Millisecond)
		}
	}

	// 7. Training loop
	fmt.Println("🚀 Starting Swarm Training Loop...")

	// Poll physics state — test10 previously never read positions back, so RL saw frozen agents.
	go func() {
		pollTicker := time.NewTicker(50 * time.Millisecond)
		defer pollTicker.Stop()
		pollBuf := make([]byte, 65536)
		for range pollTicker.C {
			writePacket(conn, []byte(`{"type":"query_constructs"}`))
			_ = conn.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
			n, err := conn.Read(pollBuf)
			_ = conn.SetReadDeadline(time.Time{})
			if err != nil {
				continue
			}
			var response struct {
				Type       string `json:"type"`
				Constructs []struct {
					ID    string `json:"id"`
					Parts []struct {
						ID  string  `json:"id"`
						Pos Vector3 `json:"pos"`
						Rot Vector3 `json:"rot"`
					} `json:"parts"`
				} `json:"constructs"`
			}
			if json.Unmarshal(pollBuf[:n], &response) != nil {
				continue
			}
			for _, construct := range response.Constructs {
				for _, a := range agents {
					if a == nil || a.ID != construct.ID {
						continue
					}
					for _, part := range construct.Parts {
						if part.ID == "core" {
							a.CurrentPos = part.Pos
							a.Rotation = part.Rot
							break
						}
					}
					break
				}
			}
		}
	}()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// Online fine-tune via loom CPU-MC (expert torque labels from live physics state).
	learningRate := float32(0.002)
	epsilon := float32(0.3)
	epsilonDecay := float32(0.995)
	epsilonMin := float32(0.05)
	torqueScale := float32(80.0)
	thrustScale := float32(10.0)

	tickCount := 0
	episode := 0
	trainSteps := 0
	lastLoss := float64(0)
	episodeReward := float32(0)

	for range ticker.C {
		tickCount++

		// Collect experiences from all agents
		updates := []UpdateRequest{}
		tickReward := float32(0)

		// Collect all agent states into single input (1500 features)
		allStates := make([]float32, 0, numAgents*inputSize)
		for _, a := range agents {
			// Calculate velocity
			dt := float32(0.05) // 50ms per tick
			a.Velocity = VecMul(VecSub(a.CurrentPos, a.LastPos), 1.0/dt)
			a.LastPos = a.CurrentPos

			state := a.GetState()
			allStates = append(allStates, state...)
		}

		// Single forward pass for entire swarm (1500 in → 300 out)
		allOutputs, _ := network.Forward(allStates)

		// Process each agent's action
		for i, a := range agents {
			// Epsilon-greedy action selection
			var action Vector3
			if rand.Float32() < epsilon {
				// Random exploration
				action = Vector3{
					rand.Float32()*2 - 1,
					rand.Float32()*2 - 1,
					rand.Float32()*2 - 1,
				}
			} else {
				outputOffset := i * outputSize
				action = Vector3{
					float32(math.Tanh(float64(allOutputs[outputOffset+0]))),
					float32(math.Tanh(float64(allOutputs[outputOffset+1]))),
					float32(math.Tanh(float64(allOutputs[outputOffset+2]))),
				}
			}

			// Apply action: torque for orientation + tangent thrust for movement.
			// Construct parts use zero gravity — spinning in place is all torque gives you.
			torque := Mul(action, torqueScale)
			moveDir := tangentToward(a.CurrentPos, a.PlanetCenter, a.TargetPos)
			align := max(0, VecDot(a.GetForward(), VecNorm(VecSub(a.TargetPos, a.CurrentPos))))
			thrust := VecMul(moveDir, thrustScale*(0.25+0.75*align))
			updates = append(updates, UpdateRequest{
				Type:        "update_construct",
				ConstructID: a.ID,
				Updates: []PartUpdate{
					{PartID: "core", Torque: &torque, LinearVelocity: &thrust},
				},
			})

			reward := calculateReward(a)
			tickReward += reward
			a.TotalReward += reward
			a.StepCount++
		}

		episodeReward += tickReward

		// Send all updates to server
		for _, u := range updates {
			d, _ := json.Marshal(u)
			writePacket(conn, d)
		}

		// Online training: supervised expert torques on current states (loom TrainOneBatch).
		if tickCount%4 == 0 {
			expertTargets := make([]float32, 0, numAgents*outputSize)
			for _, a := range agents {
				fwd := a.GetForward()
				targetDir := VecNorm(VecSub(a.TargetPos, a.CurrentPos))
				t := expertTorqueAction(fwd, targetDir)
				expertTargets = append(expertTargets, t[0], t[1], t[2])
			}
			loss, err := network.TrainOneBatch(allStates, expertTargets, learningRate)
			if err == nil {
				lastLoss = loss
				trainSteps++
			}
			if epsilon*epsilonDecay > epsilonMin {
				epsilon = epsilon * epsilonDecay
			}
		}

		// Episode metrics every 50 steps (~2.5s)
		if tickCount%50 == 0 {
			episode++
			avgReward := episodeReward / float32(numAgents)

			metrics := TrainingMetrics{
				Episode:   episode,
				AvgReward: avgReward,
				Loss:      float32(lastLoss),
				Epsilon:   epsilon,
				Timestep:  tickCount,
			}
			metricsHistory = append(metricsHistory, metrics)
			saveMetrics("swarm_training_log.csv", metricsHistory)

			fmt.Printf("📊 Episode %d - Avg Reward: %.3f - Train Loss: %.6f - Epsilon: %.3f - Steps: %d\n",
				episode, avgReward, lastLoss, epsilon, tickCount)
			episodeReward = 0
		} else if tickCount%25 == 0 {
			avgReward := episodeReward / float32(numAgents)
			fmt.Printf("🔄 Tick %d - Episode Reward: %.3f - Train Loss: %.6f - Epsilon: %.3f\n",
				tickCount, avgReward, lastLoss, epsilon)
		}

		// Save checkpoint every 500 steps
		if tickCount%500 == 0 {
			filename := fmt.Sprintf("swarm_model_checkpoint_%04d.bin", episode)
			modelID := fmt.Sprintf("swarm_ep_%d", episode)
			if err := network.SaveModel(filename, modelID); err == nil {
				fmt.Printf("💾 Saved checkpoint: %s\n", filename)
			} else {
				fmt.Printf("⚠️  Failed to save checkpoint: %v\n", err)
			}
		}
	}
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
