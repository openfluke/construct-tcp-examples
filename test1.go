package main

// Run with the whole package (types live in shared.go):
//
//	go run . 1
//
// Do not: go run test1.go

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"time"
)

// --- Test 1 Logic ---

func RunTest1() {
	conn, err := net.Dial("tcp", ConstructServerAddress)
	if err != nil {
		fmt.Printf("❌ Failed to connect to Construct Server: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("🚀 Connected to Construct Server on Port 17000")

	// Many construct hosts use a local Jolt sandbox (~±200 m floor at origin), not the
	// full multiverse planet renderer. Use meters near origin here. HUD grid cells are
	// indices; world center would be PlanetCellWorldCenter(cellX, cellY, cellZ) — often
	// far outside a small dev floor. Use a host that renders at your spawn scale.
	basePos := Vector3{0, 4, 0}

	// Step 1: Create the skeleton structure
	createReq := ConstructRequest{
		Type:        "create_construct",
		ConstructID: "client_controlled_skeleton",
		Parts: []Part{
			// Torso
			{ID: "torso", Type: "capsule", Size: Vector3{0.5, 1.2, 0}, Pos: Vector3{basePos[0], basePos[1] + 1.2, basePos[2]}, Color: Vector3{0, 1, 1}, Groups: []string{"lasso_target"}},

			// Head
			{ID: "head", Type: "capsule", Size: Vector3{0.45, 0.9, 0}, Pos: Vector3{basePos[0], basePos[1] + 2.3, basePos[2]}, Color: Vector3{0.98, 0.92, 0.84}, Groups: []string{"lasso_target"}},

			// Arms
			{ID: "l_upper", Type: "capsule", Size: Vector3{0.22, 0.8, 0}, Pos: Vector3{basePos[0] - 0.9, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, Groups: []string{"lasso_target"}, IsHorizontal: true},
			{ID: "l_fore", Type: "capsule", Size: Vector3{0.18, 0.7, 0}, Pos: Vector3{basePos[0] - 1.8, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0, 0, 0}, Groups: []string{"lasso_target"}, IsHorizontal: true},
			{ID: "r_upper", Type: "capsule", Size: Vector3{0.22, 0.8, 0}, Pos: Vector3{basePos[0] + 0.9, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, Groups: []string{"lasso_target"}, IsHorizontal: true},
			{ID: "r_fore", Type: "capsule", Size: Vector3{0.18, 0.7, 0}, Pos: Vector3{basePos[0] + 1.8, basePos[1] + 1.7, basePos[2]}, Color: Vector3{0, 0, 0}, Groups: []string{"lasso_target"}, IsHorizontal: true},

			// Legs
			{ID: "l_thigh", Type: "capsule", Size: Vector3{0.28, 0.9, 0}, Pos: Vector3{basePos[0] - 0.45, basePos[1] + 0.6, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, Groups: []string{"lasso_target"}},
			{ID: "l_shin", Type: "capsule", Size: Vector3{0.22, 0.9, 0}, Pos: Vector3{basePos[0] - 0.45, basePos[1] - 0.3, basePos[2]}, Color: Vector3{0, 0, 0}, Groups: []string{"lasso_target"}},
			{ID: "r_thigh", Type: "capsule", Size: Vector3{0.28, 0.9, 0}, Pos: Vector3{basePos[0] + 0.45, basePos[1] + 0.6, basePos[2]}, Color: Vector3{0.44, 0.5, 0.56}, Groups: []string{"lasso_target"}},
			{ID: "r_shin", Type: "capsule", Size: Vector3{0.22, 0.9, 0}, Pos: Vector3{basePos[0] + 0.45, basePos[1] - 0.3, basePos[2]}, Color: Vector3{0, 0, 0}, Groups: []string{"lasso_target"}},
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

	fmt.Println("✨ Skeleton spawned. Starting animation loop...")

	// Step 2: Animation loop
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	phase := 0.0
	for range ticker.C {
		phase += 0.05

		// Simple breathing/bobbing for torso
		bob := float32(math.Sin(phase) * 0.1)
		torsoPos := Vector3{basePos[0], basePos[1] + 1.2 + bob, basePos[2]}

		// Procedural arm swaying
		lArmTorque := Vector3{0, 0, float32(math.Sin(phase*2) * 50)}
		rArmTorque := Vector3{0, 0, float32(math.Cos(phase*2) * 50)}

		updateReq := UpdateRequest{
			Type:        "update_construct",
			ConstructID: "client_controlled_skeleton",
			Updates: []PartUpdate{
				{PartID: "torso", Position: &torsoPos},
				{PartID: "l_fore", Torque: &lArmTorque},
				{PartID: "r_fore", Torque: &rArmTorque},
			},
		}

		uData, _ := json.Marshal(updateReq)
		writePacket(conn, uData)
	}
}
