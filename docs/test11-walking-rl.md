# Test 11: Walking Skeleton Reinforcement Learning

## Overview

**Test 11** implements a **distributed multi-agent reinforcement learning system** where 20 humanoid skeleton creatures learn to walk upright using deep neural networks and policy gradient methods. This builds on the swarm RL concepts from Test 10but applies them to bipedal locomotion.

## Technical Classification

### Machine Learning Paradigm
- **Reinforcement Learning (RL)**: Agents learn locomotion through trial-and-error
- **Deep Reinforcement Learning**: Neural network policy for joint control
- **Multi-Agent Learning**: 20 skeletons share a single policy network
- **Continuous Control**: 8-dimensional continuous action space (joint torques)
- **Batched Processing**: All skeletons processed simultaneously

### Training Method
- **Policy Gradient**: Direct optimization of walking policy
- **Experience Replay**: Decorrelates temporal dependencies in walking data
- **Epsilon-Greedy Exploration**: Balances random movements vs learned gaits
- **Supervised Pre-training**: Initializes with cyclic walking patterns (CPG-like)

## System Architecture

### Neural Network

**Type**: Feedforward Deep Neural Network (Multi-Layer Perceptron)

**Architecture**:
```
Input Layer:    20 neurons (state features)
Hidden Layer 1: 128 neurons (ScaledReLU activation)
Hidden Layer 2: 64 neurons (ScaledReLU activation)
Output Layer:   8 neurons (Tanh activation - joint torques)
```

**Batch Processing**: 20 skeletons × 20 features = 400 inputs processed per forward pass

### State Representation (20 features per skeleton)

Each skeleton observes:
1. **Torso Position** (3): `x, y, z` coordinates in world space
2. **Torso Rotation** (3): `pitch, yaw, roll` Euler angles
3. **Linear Velocity** (3): `vx, vy, vz` torso movement speed
4. **Angular Velocity** (3): `ωx, ωy, ωz` torso rotational speed
5. **Joint Angles** (4): Left leg, right leg, left arm, right arm angles
6. **Forward Direction** (2): 2D heading vector (cos/sin of yaw)
7. **Height** (1): Vertical distance from spawn point
8. **Distance Traveled** (1): Total forward progress

### Action Space (8 outputs per skeleton)

**Continuous Joint Control**: Torque commands for 8 degrees of freedom

Output assignments:
1. `left_hip_forward`: Forward/backward leg swing
2. `left_hip_side`: Hip abduction/adduction
3. `left_knee`: Knee flexion/extension
4. `right_hip_forward`: Forward/backward leg swing
5. `right_hip_side`: Hip abduction/adduction  
6. `right_knee`: Knee flexion/extension
7. `left_arm`: Arm swing (balance)
8. `right_arm`: Arm swing (balance)

**Output Range**: `[-1, 1]` (Tanh activation)  
**Physical Scaling**: Multiplied by 30 for actual torque application

## Skeleton Morphology

### Body Structure (Articulated Rigid Body)

**Parts** (10 capsules):
- **Torso**: Main body (0.5×1.2m)
- **Head**: Connected via pin joint
- **Upper Arms** (2): Shoulder to elbow
- **Forearms** (2): Elbow to hand
- **Thighs** (2): Hip to knee
- **Shins** (2): Knee to foot

**Joints** (8 pin constraints):
- Neck (torso-head)
- Shoulders (2 × torso-upper arm)
- Elbows (2 × upper arm-forearm)
- Hips (2 × torso-thigh)
- Knees (2 × thigh-shin)

### Physics Properties
- **Material**: Rigid capsules with collision
- **Constraints**: Pin joints allow rotation but constrain position
- **Gravity**: Pulls skeleton downward
- **Friction**: Ground contact for foot traction

## Training Process

### Phase 1: Supervised Pre-training

**Purpose**: Initialize network with basic cyclic walking pattern

**Method**: Central Pattern Generator (CPG) simulation
- Generate 2000 synthetic walking cycles
- Simple sinusoidal leg patterns: `left = sin(phase)`, `right = sin(phase + π)`
- Arm swing opposite to legs for balance
- Train for 15 epochs using MSE loss

**Result**: Network learns basic rhythmic coordination

### Phase 2: Reinforcement Learning

**Algorithm**: Policy Gradient with Experience Replay

**Training Loop** (100 ticks per episode, 20ms per tick):
1. **State Collection**:
   - Observe torso position, rotation, velocities
   - Compute joint angles (if trackable)
   - Calculate height and distance metrics

2. **Action Selection**: Epsilon-greedy (ε=0.5 → 0.1)
   - Exploration: Random joint torques
   - Exploitation: Network policy output

3. **Physics Simulation**: Apply torques via pin joints

4. **Reward Calculation**:
   ```
   reward = forward_velocity_reward + height_reward + stability_bonus + distance_bonus
   
   forward_velocity_reward = velocity_z × 5.0
   height_reward = upright_bonus if |height - target| < 0.5 else -fall_penalty
   stability_bonus = 0.5 if angular_velocity < 1.0
   distance_bonus = total_distance × 0.1
   ```

5. **Experience Storage**: Store (state, action, reward) in replay buffer (capacity: 20,000)

6. **Network Update** (every 2 ticks if buffer ≥ 64 samples):
   - Sample minibatch of 64 experiences
   - Forward pass → predict joint torques
   - Compute loss: MSE between predicted and advantage-adjusted actions
   - Target = `action + advantage × 0.05`
   - Backward pass → update weights
   - Learning rate: 0.005

7. **Epsilon Decay**: `ε = ε × 0.995` (until reaching 0.1 minimum)

## Reward Function Breakdown

### 1. Forward Movement (Primary Objective)
```go
forwardVel := velocity[2] // Z-axis
reward += forwardVel × 5.0
```
Encourages forward locomotion (main task).

### 2. Upright Posture
```go
heightDiff := abs(torsoHeight - targetHeight)
if heightDiff < 0.5 {
    heightReward = 1.0 - heightDiff  // Bonus for staying upright
} else {
    heightReward = -heightDiff // Penalty for falling
}
reward += heightReward × 2.0
```
Prevents falling down (bipedal stability).

### 3. Stability Bonus
```go
angularMag := sqrt(ωx² + ωy² + ωz²)
if angularMag < 1.0 {
    reward += 0.5
}
```
Rewards smooth, stable movement over erratic flailing.

### 4. Distance Bonus
```go
reward += total_distance_traveled × 0.1
```
Long-term progress incentive.

## Locomotion Challenges

### Bipedal Walking is Hard!
Walking requires:
1. **Balance**: Maintaining upright posture against gravity
2. **Coordination**: Synchronizing leg and arm movements
3. **Stability**: Preventing falls during weight transfer
4. **Efficiency**: Minimizing energy (torque) expenditure
5. **Rhythm**: Discovering periodic gait patterns

### Expected Learning Curve
- **Episodes 0-20**: Random flailing, frequent falls
- **Episodes 20-50**: Discovering balance, some forward drift
- **Episodes 50-100**: Emerging gait patterns, consistent forward movement
- **Episodes 100+**: Refined walking, possibly running gaits

## Key Implementation Details

### Velocity Calculation
```go
dt = 0.02 // 20ms per tick
velocity = (currentTorsoPos - lastTorsoPos) / dt
```

### Batched Forward Pass
```go
allStates = [skeleton0_state(20), skeleton1_state(20), ..., skeleton19_state(20)]
allOutputs = network.Forward(allStates) // 400 in → 160 out
```

### Joint Torque Application
```go
for i, skeleton := range skeletons {
    outputOffset := i * 8
    leftHipTorque = Vector3{
        outputs[outputOffset + 0] * 30,  // Forward/back
        0,
        outputs[outputOffset + 1] * 30,  // Side
    }
    // Apply to l_thigh part via UpdateRequest
}
```

## Differences from Test 10 (Swarm RL)

| Aspect | Test 10 (Cubes) | Test 11 (Skeletons) |
|---|---|---|
| **Agents** | 100 simple cubes | 20 articulated skeletons |
| **State Size** | 15 features | 20 features |
| **Action Size** | 3 torques | 8 torques |
| **Complexity** | Rotation control only | Full locomotion |
| **Task** | Navigate to target | Walk forward upright |
| **Reward** | Alignment + exploration | Forward + balance |
| **Episode** | 50 ticks (1s) | 100 ticks (2s) |
| **Difficulty** | Moderate | High |

## Related Concepts

**This system combines**:
- Bipedal Locomotion Control
- Multi-Agent RL (MARL)
- Continuous Action Spaces
- Physics-Based Animation
- Central Pattern Generators (CPG)
- Policy Gradient Methods

**Similar To**:
- DeepMind's MuJoCo humanoid walker
- OpenAI Gym Walker2D/BipedalWalker
- Evolution Strategies for walking gaits  
- Reinforcement Learning for prosthetic control
- Boston Dynamics robot learning

## Future Enhancements

Potential improvements:
1. **Curriculum Learning**: Start with crawling, progress to walking/running
2. **Terrain Adaptation**: Hills, obstacles, varying surfaces
3. **Multi-Objective**: Walking + carrying objects
4. **Inverse Kinematics**: Target foot placement
5. **Imitation Learning**: Learn from motion capture data
6. **Adversarial Training**: Push-recovery, navigation under

 disturbances
7. **Hierarchical Control**: High-level gait selection + low-level joint control
8. **Evolution Strategies**: Explore morphology variations

## Technical Stack

- **Language**: Go
- **ML Framework**: Custom `loom/nn` neural network library
- **Physics**: Server-side articulated rigid body dynamics via WebSocket
- **Training**: CPU-based backpropagation
- **Constraints**: Pin joints for skeletal structure
- **Serialization**: Binary checkpoint format

## Performance Metrics

### Training Indicators
- **Average Reward**: Mean reward across 20 skeletons per episode
- **Epsilon**: Current exploration rate
- **Buffer Size**: Experiences stored (max 20,000)
- **Forward Distance**: How far skeleton traveled

### Success Criteria
- **Walking**: Consistent forward velocity > 0.5 m/s
- **Stability**: Torso height maintained within 20% of target
- **Episodes Without Falls**: Count before tumbling

### Checkpointing
- Model saved every 10 episodes
- Format: `walking_skeleton_checkpoint_XXXX.bin`
- Includes network weights and optimizer state

---

**Author**: Walking Skeleton RL System  
**Date**: 2026-02-04  
**Version**: 1.0 - Distributed Multi-Agent Bipedal Locomotion  
**Based On**: Test 10 Swarm RL Architecture
