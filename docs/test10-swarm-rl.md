# Test 10: Swarm Reinforcement Learning

## Overview

**Test 10** implements a **distributed multi-agent reinforcement learning system** where 100 autonomous cube agents learn to navigate a 3D planetary environment using deep neural networks and policy gradient methods.

## Technical Classification

### Machine Learning Paradigm
- **Reinforcement Learning (RL)**: Agents learn through trial-and-error interaction with the environment
- **Deep Reinforcement Learning**: Uses neural networks as function approximators
- **Multi-Agent Learning**: 100 agents share a single policy network
- **Batched Processing**: All agents processed simultaneously for efficiency

### Training Method
- **Policy Gradient**: Direct optimization of the policy (action selection)
- **Experience Replay**: Decorrelates samples by storing and sampling past experiences
- **Epsilon-Greedy Exploration**: Balances exploitation vs exploration
- **Supervised Pre-training**: Initializes network with expert demonstrations

## System Architecture

### Neural Network

**Type**: Feedforward Deep Neural Network (Multi-Layer Perceptron)

**Architecture**:
```
Input Layer:    15 neurons (state features)
Hidden Layer 1: 64 neurons (ScaledReLU activation)
Hidden Layer 2: 32 neurons (ScaledReLU activation)
Output Layer:   3 neurons (Tanh activation)
```

**Batch Processing**: 100 agents × 15 features = 1500 inputs processed per forward pass

### State Representation (15 features per agent)

Each agent observes:
1. **Position** (3): `x, y, z` coordinates in world space
2. **Rotation** (3): `pitch, yaw, roll` Euler angles
3. **Linear Velocity** (3): `vx, vy, vz` movement speed
4. **Angular Velocity** (3): `ωx, ωy, ωz` rotational speed  
5. **Target Position** (3): `target_x, target_y, target_z` goal location

### Action Space (3 outputs per agent)

**Continuous Control**: Torque commands for 3-axis rotation, plus automatic tangent-plane thrust.

- `pitch_torque`, `yaw_torque`, `roll_torque` — network outputs (Tanh, scaled ×100)
- **`linear_velocity`** — applied each tick toward the next bubble along the planet tangent plane (construct sim is **zero gravity**; torque alone only spins cubes in place)

**Output Range**: `[-1, 1]` (Tanh activation) for torques  
**Thrust**: up to ~10 m/s toward target when aligned (blended with alignment factor)

### State sync

Each tick, a background poll sends **`query_constructs`** and updates each agent's `CurrentPos` / `Rotation` from the physics server. Without this, the RL loop optimizes against frozen spawn positions and never sees movement.

## Training Process

Training uses **loom/poly `Train()` on CPU-MC** (`TrainingModeCPUMC` — multi-core tiled CPU backward), not the old hand-rolled forward/backward loop.

### Phase 1: Supervised Pre-training

- 5000 expert demos with **normalized** 15-D state (~[-1, 1])
- Expert torque targets in **[-1, 1]** (linear output layer; `tanh` applied at action time)
- Batched MSE via `poly.Train` — loss should **decrease each epoch** (see `go test -run TestPretrainLossDecreases`)

### Phase 2: Online fine-tuning (live loop)

Every 4 simulation ticks:

1. Poll physics via `query_constructs`
2. Build normalized states for all agents
3. Label with **expert torque** from current forward vs target direction
4. One `TrainOneBatch` step (loom CPU-MC)

Epsilon-greedy exploration still randomizes actions; the network learns to imitate the expert on live planet states.

### Metrics

- **Pre-train:** JSON lines `{"epoch":N,"loss":…}` — loss should fall
- **Live:** `Train Loss:` in tick/episode logs — should trend down while cubes align and move

### Technical Terms Explained

**Experience Replay Buffer**: 
- Stores past (state, action, reward) tuples
- Breaks temporal correlation in training data
- Allows repeated use of rare experiences

**Policy Gradient**:
- Directly optimizes the policy function `π(a|s)`
- Adjusts policy in direction of higher rewards
- Uses advantage function to amplify good actions

**Batched Inference**:
- Processes all 100 agents in one forward pass
- GPU/CPU parallelization (100x speedup vs sequential)
- Shared weights across all agents

**Epsilon-Greedy**:
- ε probability of random action (exploration)
- (1-ε) probability of network action (exploitation)
- Balances learning new strategies vs using known good actions

## Swarm Characteristics

### Distributed Learning
- **Single Shared Policy**: All 100 agents use the same neural network
- **Collective Experience**: Each agent contributes to shared replay buffer
- **Emergent Behavior**: Network learns generalizable navigation strategy

### Data Efficiency
- 100 agents × 50 ticks/episode = 5000 experiences per episode
- Faster learning compared to single-agent RL
- Diverse situations encountered simultaneously

## Performance Metrics

### Training Indicators
- **Average Reward**: Mean reward across all agents per episode
- **Epsilon**: Current exploration rate
- **Buffer Size**: Number of experiences stored
- **Loss**: Training loss (MSE between predicted and target actions)

### Checkpointing
- Model saved every 10 episodes
- Format: `swarm_model_checkpoint_XXXX.bin`
- Includes network weights and optimizer state

## Physical Environment

### Planetary Navigation
- **Planet**: Spherical body with gravity
- **Bubbles**: 10 target waypoints distributed on surface
- **Spawn Locations**: Agents spawn in rings around bubbles
- **Task**: Navigate from current bubble to next bubble

### Physics Integration
- **Engine**: Server-side rigid body physics (**zero gravity** on construct parts)
- **Control**: Torque for orientation + **`linear_velocity` thrust** tangent to the planet toward the next bubble
- **Constraints**: Cubes must be polled via `query_constructs` for RL state; thrust provides translation

## Key Implementation Details

### Velocity Calculation
```go
dt = 0.05 // 50ms per tick
velocity = (currentPos - lastPos) / dt
```

### Batched Forward Pass
```go
allStates = [agent0_state(15), agent1_state(15), ..., agent99_state(15)]
allOutputs = network.Forward(allStates) // 1500 in → 300 out
```

### Output Extraction
```go
for i, agent := range agents {
    outputOffset := i * 3
    action = Vector3{
        allOutputs[outputOffset + 0], // pitch
        allOutputs[outputOffset + 1], // yaw  
        allOutputs[outputOffset + 2], // roll
    }
}
```

## Related Concepts

**This system combines**:
- Multi-Agent Reinforcement Learning (MARL)
- Deep Q-Learning principles (experience replay)
- Policy Gradient methods (direct policy optimization)
- Behavioral Cloning (pre-training)
- Batch Learning (efficient parallel processing)

**Similar To**:
- OpenAI's multi-agent hide-and-seek
- DeepMind's StarCraft II learning
- Robotic swarm coordination
- Distributed ML training

## Future Enhancements

Potential improvements:
1. **Attention Mechanisms**: Allow agents to observe nearby agents
2. **Hierarchical Policies**: High-level strategy + low-level control
3. **Curriculum Learning**: Progressively harder navigation tasks
4. **Mixed Objectives**: Multiple simultaneous goals
5. **Communication Protocols**: Explicit agent-to-agent messages
6. **Adversarial Training**: Competitive multi-agent scenarios

## Technical Stack

- **Language**: Go
- **ML**: loom/poly **CPU-MC** (`ConfigureNetworkForMode` + `poly.Train`)
- **Physics**: Construct TCP server (zero-G parts + client thrust)
- **Checkpoints**: `swarm_model_checkpoint_XXXX.bin`

---

**Author**: Swarm RL Test Suite  
**Date**: 2026-02-04  
**Version**: 1.0 - Batched Feedforward Architecture
