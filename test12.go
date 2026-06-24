package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"time"
)

// --- Test 12: Swarm Skeletons ---

func RunTest12() {
	fmt.Println("🦴 Starting Test 12: SWARM SKELETONS 🦴")

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

	fmt.Printf("✅ Found %d bubbles. Spawning %d skeletons...\n", numBubbles, numSkeletons)

	planetCenter := Vector3{state.PlanetCenter[0], state.PlanetCenter[1], state.PlanetCenter[2]}
	spawnOffset := float32(3.0) // Offset above bubble surface

	skeletonIdx := 0
	for i := 0; i < numBubbles; i++ {
		b := state.Bubbles[i]
		bPos := Vector3{b.Pos[0], b.Pos[1], b.Pos[2]}
		up := VecNorm(VecSub(bPos, planetCenter))

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

			id := fmt.Sprintf("skeleton_12_%d", skeletonIdx)

			// Spawn skeleton
			createSkeletonGrabbable(conn, id, spawnPos)
			skeletonIdx++
			time.Sleep(20 * time.Millisecond) // Slight delay to avoid flooding
		}
	}

	fmt.Println("✅ All skeletons spawned. Staying connected...")
	// Keep connection alive or just exit? Skeletons remain on server.
	// But usually these tests stay alive for a bit.
	select {}
}

func createSkeletonGrabbable(conn net.Conn, id string, basePos Vector3) {
	groups := []string{"lasso_target", "skeleton"}

	createReq := ConstructRequest{
		Type:        "create_construct",
		ConstructID: id,
		Parts: []Part{
			// Torso
			{ID: "torso", Type: "capsule", Size: Vector3{0.5, 1.2, 0}, Pos: Vector3{basePos[0], basePos[1] + 1.2, basePos[2]}, Color: Vector3{0.9, 0.7, 0.5}, Groups: groups},

			// Head
			{ID: "head", Type: "capsule", Size: Vector3{0.45, 0.9, 0}, Pos: Vector3{basePos[0], basePos[1] + 2.3, basePos[2]}, Color: Vector3{0.98, 0.92, 0.84}, Groups: groups},

			// Arms
			{ID: "l_upper", Type: "capsule", Size: Vector3{0.22, 0.8, 0}, Pos: Vector3{basePos[0] - 0.9, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, IsHorizontal: true, Groups: groups},
			{ID: "l_fore", Type: "capsule", Size: Vector3{0.18, 0.7, 0}, Pos: Vector3{basePos[0] - 1.8, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.3, 0.3, 0.3}, IsHorizontal: true, Groups: groups},
			{ID: "r_upper", Type: "capsule", Size: Vector3{0.22, 0.8, 0}, Pos: Vector3{basePos[0] + 0.9, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, IsHorizontal: true, Groups: groups},
			{ID: "r_fore", Type: "capsule", Size: Vector3{0.18, 0.7, 0}, Pos: Vector3{basePos[0] + 1.8, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.3, 0.3, 0.3}, IsHorizontal: true, Groups: groups},

			// Legs
			{ID: "l_thigh", Type: "capsule", Size: Vector3{0.28, 0.9, 0}, Pos: Vector3{basePos[0] - 0.45, basePos[1] + 0.6, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, Groups: groups},
			{ID: "l_shin", Type: "capsule", Size: Vector3{0.22, 0.9, 0}, Pos: Vector3{basePos[0] - 0.45, basePos[1] - 0.3, basePos[2]}, Color: Vector3{0.3, 0.3, 0.3}, Groups: groups},
			{ID: "r_thigh", Type: "capsule", Size: Vector3{0.28, 0.9, 0}, Pos: Vector3{basePos[0] + 0.45, basePos[1] + 0.6, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, Groups: groups},
			{ID: "r_shin", Type: "capsule", Size: Vector3{0.22, 0.9, 0}, Pos: Vector3{basePos[0] + 0.45, basePos[1] - 0.3, basePos[2]}, Color: Vector3{0.3, 0.3, 0.3}, Groups: groups},
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
