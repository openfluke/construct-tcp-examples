package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"time"
)

// --- Test 11: Walking Skeleton RL ---

// SkeletonAgent represents one learning skeleton creature
type SkeletonAgent struct {
	ID              string
	SpawnPos        Vector3
	TorsoPos        Vector3
	LastTorsoPos    Vector3
	Velocity        Vector3
	TorsoRotation   Vector3
	AngularVelocity Vector3
	TargetPos       Vector3 // Target bubble position to navigate to

	// Joint states (for observation)
	LeftLegAngle  float32
	RightLegAngle float32
	LeftArmAngle  float32
	RightArmAngle float32

	// Learning metrics
	TotalReward      float32
	StepCount        int
	DistanceTraveled float32
}

func (s *SkeletonAgent) GetState() []float32 {
	// Rich 23-feature state representation
	state := []float32{
		// Torso state (9)
		s.TorsoPos[0], s.TorsoPos[1], s.TorsoPos[2],
		s.TorsoRotation[0], s.TorsoRotation[1], s.TorsoRotation[2],
		s.Velocity[0], s.Velocity[1], s.Velocity[2],

		// Angular state (3)
		s.AngularVelocity[0], s.AngularVelocity[1], s.AngularVelocity[2],

		// Joint angles (4)
		s.LeftLegAngle, s.RightLegAngle,
		s.LeftArmAngle, s.RightArmAngle,

		// Forward direction to goal (2D heading)
		float32(math.Cos(float64(s.TorsoRotation[1]) * math.Pi / 180.0)),
		float32(math.Sin(float64(s.TorsoRotation[1]) * math.Pi / 180.0)),

		// Height above ground
		s.TorsoPos[1],

		// Distance traveled so far		float32(s.DistanceTraveled),
		float32(s.DistanceTraveled),
	}

	// Add target position relative to current position (3 features)
	relativeTarget := VecSub(s.TargetPos, s.TorsoPos)
	state = append(state, relativeTarget[0], relativeTarget[1], relativeTarget[2])

	return state
}

func RunTest11() {
	fmt.Println("🦴 Starting Test 11: WALKING SKELETON RL 🦴")

	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ Failed to connect to Construct Server: %v\n", err)
		return
	}
	defer conn.Close()

	// Query planet state for bubbles
	fmt.Println("📡 Querying world state for bubbles...")
	writePacket(conn, []byte(`{"type":"query_state"}`))

	buf := make([]byte, 32768)
	n, _ := conn.Read(buf)
	var state StateResponse
	json.Unmarshal(buf[:n], &state)

	if len(state.Bubbles) == 0 {
		fmt.Println("❌ No bubbles found in world state")
		return
	}

	numBubbles := len(state.Bubbles)
	skeletonsPerBubble := 3
	numSkeletons := numBubbles * skeletonsPerBubble

	fmt.Printf("✅ Found %d bubbles. Initializing %d skeletons...\n", numBubbles, numSkeletons)

	// Configuration
	inputSize := 23 // Extended state: position, rotation, velocities, target, bubble info
	outputSize := 8 // 8 joint torques

	planetCenter := Vector3{state.PlanetCenter[0], state.PlanetCenter[1], state.PlanetCenter[2]}

	// Create neural network
	fmt.Println("🏗️  Building Walking Neural Network...")
	fmt.Printf("   - Input: %d features per skeleton\n", inputSize)
	fmt.Printf("   - Architecture: Dense(%d → 128 → 64 → %d)\n", inputSize, outputSize)
	fmt.Printf("   - Batch size: %d skeletons processed together\n", numSkeletons)

	network := NewNetwork(inputSize, 1, 3, 1)
	network.BatchSize = numSkeletons

	layer0 := InitDenseLayer(inputSize, 128, ActivationScaledReLU)
	layer1 := InitDenseLayer(128, 64, ActivationScaledReLU)
	layer2 := InitDenseLayer(64, outputSize, ActivationTanh)

	network.SetLayer(0, 0, 0, layer0)
	network.SetLayer(0, 1, 0, layer1)
	network.SetLayer(0, 2, 0, layer2)
	network.GPU = false

	fmt.Println("✅ Network initialized")

	// Pre-training
	preTrainWalkingNetwork(network, inputSize, outputSize, numSkeletons)

	// Spawn skeletons around bubbles (like test10)
	skeletons := make([]*SkeletonAgent, numSkeletons)
	spawnOffset := float32(3.0) // Offset above bubble surface

	fmt.Printf("🚀 Spawning %d skeletons across %d bubbles...\n", numSkeletons, numBubbles)

	skeletonIdx := 0
	for i := 0; i < numBubbles; i++ {
		b := state.Bubbles[i]
		bPos := Vector3{b.Pos[0], b.Pos[1], b.Pos[2]}
		up := VecNorm(VecSub(bPos, planetCenter))

		// Target: next bubble in sequence
		nextIdx := (i + 1) % numBubbles
		targetBubble := state.Bubbles[nextIdx]
		targetPos := Vector3{targetBubble.Pos[0], targetBubble.Pos[1], targetBubble.Pos[2]}

		// Spawn skeletons in ring around bubble
		for j := 0; j < skeletonsPerBubble; j++ {
			theta := (float64(j) / float64(skeletonsPerBubble)) * 2.0 * math.Pi
			ringDist := float32(8.0)
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

			id := fmt.Sprintf("skeleton_%d", skeletonIdx)

			skeletons[skeletonIdx] = &SkeletonAgent{
				ID:           id,
				SpawnPos:     spawnPos,
				TorsoPos:     Vector3{spawnPos[0], spawnPos[1] + 1.2, spawnPos[2]},
				LastTorsoPos: Vector3{spawnPos[0], spawnPos[1] + 1.2, spawnPos[2]},
				TargetPos:    targetPos, // Navigate to next bubble!
			}

			// Spawn skeleton
			createSkeleton(conn, id, spawnPos)
			skeletonIdx++
		}
	}

	fmt.Println("🚀 Starting Swarm Walking Training Loop...")

	// Start background goroutine to poll skeleton states from server
	go func() {
		pollTicker := time.NewTicker(50 * time.Millisecond) // Poll every 50ms
		defer pollTicker.Stop()

		pollBuf := make([]byte, 65536) // Larger buffer for all construct data

		for range pollTicker.C {
			// Request full state with all constructs
			writePacket(conn, []byte(`{"type":"query_constructs"}`))

			// Read response (non-blocking with timeout would be better but keep it simple)
			conn.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
			n, err := conn.Read(pollBuf)
			if err != nil {
				continue // Skip this poll if timeout/error
			}
			conn.SetReadDeadline(time.Time{}) // Clear deadline

			// Try to parse as a generic response with parts
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

			if json.Unmarshal(pollBuf[:n], &response) == nil && len(response.Constructs) > 0 {
				// Update skeleton positions
				for _, construct := range response.Constructs {
					// Find matching skeleton
					for _, s := range skeletons {
						if s.ID == construct.ID {
							// Find torso part
							for _, part := range construct.Parts {
								if part.ID == "torso" {
									s.TorsoPos = part.Pos
									s.TorsoRotation = part.Rot

									// Update distance traveled
									dx := s.TorsoPos[0] - s.SpawnPos[0]
									dy := s.TorsoPos[1] - s.SpawnPos[1]
									dz := s.TorsoPos[2] - s.SpawnPos[2]
									dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
									if dist > s.DistanceTraveled {
										s.DistanceTraveled = dist
									}
									break
								}
							}
							break
						}
					}
				}
			}
		}
	}()

	// RL Training parameters
	learningRate := float32(0.005)
	epsilon := float32(0.5) // Start with high exploration
	epsilonMin := float32(0.1)
	epsilonDecay := float32(0.995)

	// Experience replay
	expBuffer := NewExperienceBuffer(20000)

	// Training loop
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	tickCount := 0
	episode := 0
	trainSteps := 0

	for range ticker.C {
		tickCount++

		// Collect all skeleton states
		allStates := make([]float32, 0, numSkeletons*inputSize)
		for _, s := range skeletons {
			// Calculate velocity
			dt := float32(0.02) // 20ms
			s.Velocity = VecMul(VecSub(s.TorsoPos, s.LastTorsoPos), 1.0/dt)
			s.LastTorsoPos = s.TorsoPos

			state := s.GetState()
			allStates = append(allStates, state...)
		}

		// Batched forward pass
		allOutputs, _ := network.Forward(allStates)

		// Apply actions to each skeleton
		updates := []UpdateRequest{}
		totalReward := float32(0)

		for i, s := range skeletons {
			// Extract this skeleton's action (8 torques)
			outputOffset := i * outputSize
			action := make([]float32, outputSize)

			// Epsilon-greedy
			if rand.Float32() < epsilon {
				// Random exploration
				for j := 0; j < outputSize; j++ {
					action[j] = rand.Float32()*2 - 1
				}
			} else {
				// Network policy
				for j := 0; j < outputSize; j++ {
					action[j] = allOutputs[outputOffset+j]
				}
			}

			// Scale torques
			torqueScale := float32(30.0)

			// Map actions to joint torques
			leftHipTorque := Vector3{action[0] * torqueScale, 0, action[1] * torqueScale}
			leftKneeTorque := Vector3{action[2] * torqueScale, 0, 0}
			rightHipTorque := Vector3{action[3] * torqueScale, 0, action[4] * torqueScale}
			rightKneeTorque := Vector3{action[5] * torqueScale, 0, 0}
			leftArmTorque := Vector3{0, 0, action[6] * torqueScale}
			rightArmTorque := Vector3{0, 0, action[7] * torqueScale}

			updates = append(updates, UpdateRequest{
				Type:        "update_construct",
				ConstructID: s.ID,
				Updates: []PartUpdate{
					{PartID: "l_thigh", Torque: &leftHipTorque},
					{PartID: "l_shin", Torque: &leftKneeTorque},
					{PartID: "r_thigh", Torque: &rightHipTorque},
					{PartID: "r_shin", Torque: &rightKneeTorque},
					{PartID: "l_fore", Torque: &leftArmTorque},
					{PartID: "r_fore", Torque: &rightArmTorque},
				},
			})

			// Calculate reward
			reward := calculateWalkingReward(s)
			totalReward += reward

			// Store experience
			stateOffset := i * inputSize
			exp := Experience{
				State:  allStates[stateOffset : stateOffset+inputSize],
				Action: action,
				Reward: reward,
			}
			expBuffer.Add(exp)

			s.TotalReward += reward
			s.StepCount++
		}

		// Send all updates
		for _, u := range updates {
			d, _ := json.Marshal(u)
			writePacket(conn, d)
		}

		// Note: We can't easily read back positions from server without complex state tracking
		// Instead, we rely on velocity calculations and assume physics is working
		// The skeletons will move based on applied torques

		// Training step (every 2 ticks)
		if tickCount%2 == 0 && expBuffer.Size >= 64 {
			batchSize := 64
			batch := expBuffer.Sample(batchSize)

			trainBatchStates := make([]float32, 0, batchSize*inputSize)
			trainBatchTargets := make([]float32, 0, batchSize*outputSize)

			for _, exp := range batch {
				trainBatchStates = append(trainBatchStates, exp.State...)

				// Policy gradient target
				advantage := exp.Reward
				for j := 0; j < outputSize; j++ {
					trainBatchTargets = append(trainBatchTargets, exp.Action[j]+advantage*0.05)
				}
			}

			// Train
			oldBatchSize := network.BatchSize
			network.BatchSize = batchSize

			output, _ := network.Forward(trainBatchStates)

			grad := make([]float32, len(output))
			totalLoss := float32(0)
			for j := 0; j < len(output); j++ {
				err := output[j] - trainBatchTargets[j]
				totalLoss += err * err
				grad[j] = err
			}

			network.Backward(grad)
			network.ApplyGradients(learningRate)

			network.BatchSize = oldBatchSize
			trainSteps++

			// Decay epsilon
			if epsilon*epsilonDecay > epsilonMin {
				epsilon = epsilon * epsilonDecay
			}
		}

		// Episode reset every 100 ticks (2 seconds)
		if tickCount%100 == 0 {
			episode++
			avgReward := totalReward / float32(numSkeletons)

			// Status update
			fmt.Printf("📊 Episode %d - Avg Reward: %.3f - Epsilon: %.3f - Buffer: %d\\n",
				episode, avgReward, epsilon, expBuffer.Size)

			// Save checkpoint every 10 episodes
			if episode%10 == 0 {
				filename := fmt.Sprintf("walking_skeleton_checkpoint_%04d.bin", episode)
				modelID := fmt.Sprintf("walk_ep_%d", episode)
				if err := network.SaveModel(filename, modelID); err == nil {
					fmt.Printf("💾 Saved checkpoint: %s\\n", filename)
				}
			}
		}
	}
}

func createSkeleton(conn net.Conn, id string, basePos Vector3) {
	createReq := ConstructRequest{
		Type:        "create_construct",
		ConstructID: id,
		Parts: []Part{
			// Torso
			{ID: "torso", Type: "capsule", Size: Vector3{0.5, 1.2, 0}, Pos: Vector3{basePos[0], basePos[1] + 1.2, basePos[2]}, Color: Vector3{0.9, 0.7, 0.5}},

			// Head
			{ID: "head", Type: "capsule", Size: Vector3{0.45, 0.9, 0}, Pos: Vector3{basePos[0], basePos[1] + 2.3, basePos[2]}, Color: Vector3{0.98, 0.92, 0.84}},

			// Arms
			{ID: "l_upper", Type: "capsule", Size: Vector3{0.22, 0.8, 0}, Pos: Vector3{basePos[0] - 0.9, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, IsHorizontal: true},
			{ID: "l_fore", Type: "capsule", Size: Vector3{0.18, 0.7, 0}, Pos: Vector3{basePos[0] - 1.8, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.3, 0.3, 0.3}, IsHorizontal: true},
			{ID: "r_upper", Type: "capsule", Size: Vector3{0.22, 0.8, 0}, Pos: Vector3{basePos[0] + 0.9, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, IsHorizontal: true},
			{ID: "r_fore", Type: "capsule", Size: Vector3{0.18, 0.7, 0}, Pos: Vector3{basePos[0] + 1.8, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.3, 0.3, 0.3}, IsHorizontal: true},

			// Legs
			{ID: "l_thigh", Type: "capsule", Size: Vector3{0.28, 0.9, 0}, Pos: Vector3{basePos[0] - 0.45, basePos[1] + 0.6, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}},
			{ID: "l_shin", Type: "capsule", Size: Vector3{0.22, 0.9, 0}, Pos: Vector3{basePos[0] - 0.45, basePos[1] - 0.3, basePos[2]}, Color: Vector3{0.3, 0.3, 0.3}},
			{ID: "r_thigh", Type: "capsule", Size: Vector3{0.28, 0.9, 0}, Pos: Vector3{basePos[0] + 0.45, basePos[1] + 0.6, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}},
			{ID: "r_shin", Type: "capsule", Size: Vector3{0.22, 0.9, 0}, Pos: Vector3{basePos[0] + 0.45, basePos[1] - 0.3, basePos[2]}, Color: Vector3{0.3, 0.3, 0.3}},
		},
		Joints: []Joint{
			{Type: "pin", A: "torso", B: "head", Pos: Vector3{basePos[0], basePos[1] + 2.0, basePos[2]}},
			{Type: "pin", A: "torso", B: "l_upper", Pos: Vector3{basePos[0] - 0.55, basePos[1] + 1.7, basePos[2]}},
			{Type: "pin", A: "l_upper", B: "l_fore", Pos: Vector3{basePos[0] - 1.3, basePos[1] + 1.7, basePos[2]}},
			{Type: "pin", A: "torso", B: "r_upper", Pos: Vector3{basePos[0] + 0.55, basePos[1] + 1.7, basePos[2]}},
			{Type: "pin", A: "r_upper", B: "r_fore", Pos: Vector3{basePos[0] + 1.3, basePos[1] + 1.7, basePos[2]}},
			{Type: "pin", A: "torso", B: "l_thigh", Pos: Vector3{basePos[0] - 0.45, basePos[1] + 1.1, basePos[2]}},
			{Type: "pin", A: "l_thigh", B: "l_shin", Pos: Vector3{basePos[0] - 0.45, basePos[1] + 0.1, basePos[2]}},
			{Type: "pin", A: "torso", B: "r_thigh", Pos: Vector3{basePos[0] + 0.45, basePos[1] + 1.1, basePos[2]}},
			{Type: "pin", A: "r_thigh", B: "r_shin", Pos: Vector3{basePos[0] + 0.45, basePos[1] + 0.1, basePos[2]}},
		},
	}

	data, _ := json.Marshal(createReq)
	writePacket(conn, data)
}

func calculateWalkingReward(s *SkeletonAgent) float32 {
	reward := float32(0)

	// 1. Forward movement reward (main objective)
	forwardVel := s.Velocity[2] // Z-axis is forward
	reward += forwardVel * 5.0

	// 2. Upright posture reward
	heightReward := float32(0)
	targetHeight := s.SpawnPos[1] + 1.2
	heightDiff := float32(math.Abs(float64(s.TorsoPos[1] - targetHeight)))
	if heightDiff < 0.5 {
		heightReward = 1.0 - heightDiff
	} else {
		heightReward = -heightDiff // Penalty for falling
	}
	reward += heightReward * 2.0

	// 3. Stability bonus (low angular velocity)
	angularMag := float32(math.Sqrt(float64(
		s.AngularVelocity[0]*s.AngularVelocity[0] +
			s.AngularVelocity[1]*s.AngularVelocity[1] +
			s.AngularVelocity[2]*s.AngularVelocity[2])))
	if angularMag < 1.0 {
		reward += 0.5
	}

	// 4. Energy efficiency (penalize excessive torque)
	// This is implicitly handled by the network learning efficient gaits

	// 5. Distance traveled bonus
	distBonus := s.DistanceTraveled * 0.1
	reward += distBonus

	return reward
}

func preTrainWalkingNetwork(network *Network, inputSize, outputSize, numSkeletons int) {
	fmt.Println("🛰️  Pre-Training on Walking Gaits...")

	// Generate synthetic walking data with cyclic patterns
	numSamples := numSkeletons // Match the actual number of skeletons
	inputs := make([][]float32, numSamples)
	targets := make([][]float32, numSamples)

	for i := 0; i < numSamples; i++ {
		phase := float32(i) * 0.1

		// Random starting state (23 features)
		state := make([]float32, inputSize)
		for j := 0; j < inputSize; j++ {
			state[j] = rand.Float32()*2 - 1
		}
		inputs[i] = state

		// Cyclic walking pattern (simple CPG-like)
		leftLegPhase := float32(math.Sin(float64(phase)))
		rightLegPhase := float32(math.Sin(float64(phase + math.Pi)))

		targets[i] = []float32{
			leftLegPhase * 0.5,   // Left hip forward/back
			0,                    // Left hip side
			leftLegPhase * 0.3,   // Left knee
			rightLegPhase * 0.5,  // Right hip forward/back
			0,                    // Right hip side
			rightLegPhase * 0.3,  // Right knee
			-leftLegPhase * 0.2,  // Left arm swing (opposite)
			-rightLegPhase * 0.2, // Right arm swing (opposite)
		}
	}

	config := DefaultTrainingConfig()
	config.Epochs = 15
	config.LearningRate = 0.01
	config.UseGPU = false
	config.Verbose = true
	config.LossType = "mse"

	network.TrainStandard(inputs, targets, config)

	fmt.Println("✅ Pre-Training Complete")
}
