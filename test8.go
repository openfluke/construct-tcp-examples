package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"time"
)

func RunTest8() {
	fmt.Println("🎈 Starting Test 8: Multi-Bubble Discovery & Spawning...")

	// 1. Connect and Query State
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ Connection Failed: %v\n", err)
		return
	}
	defer conn.Close()

	// Send Query
	query := map[string]string{"type": "query_state"}
	qData, _ := json.Marshal(query)
	writePacket(conn, qData)

	// Read Response
	buf := make([]byte, 32768) // Large buffer for potentially many bubbles
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("❌ Read Failed: %v\n", err)
		return
	}

	var state StateResponse
	err = json.Unmarshal(buf[:n], &state)
	if err != nil {
		fmt.Printf("❌ Parse Status Failed: %v\nMessage: %s\n", err, string(buf[:n]))
		return
	}

	if state.Type != "state_response" {
		fmt.Printf("❌ Unexpected Response: %s\n", state.Type)
		return
	}

	fmt.Printf("✅ DISCOVERY SUCCESSFUL!\n")
	fmt.Printf("   🌍 Planet: %v (Radius: %.1f)\n", state.PlanetCenter, state.PlanetRadius)
	fmt.Printf("   🫧 Bubbles Found: %d\n", len(state.Bubbles))

	if len(state.Bubbles) == 0 {
		fmt.Println("⚠️ No bubbles found. Wait for planet initialization or move closer.")
		return
	}

	planetCenter := Vector3{state.PlanetCenter[0], state.PlanetCenter[1], state.PlanetCenter[2]}

	// 2. Iterate and Spawn on/above each bubble
	for _, bubble := range state.Bubbles {
		bubblePos := Vector3{bubble.Pos[0], bubble.Pos[1], bubble.Pos[2]}

		// Calculate Up vector (normal)
		dx := bubblePos[0] - planetCenter[0]
		dy := bubblePos[1] - planetCenter[1]
		dz := bubblePos[2] - planetCenter[2]
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		up := Vector3{dx / dist, dy / dist, dz / dist}

		fmt.Printf("📍 Spawning markers for Bubble #%d\n", bubble.Index)

		// Spawn "on" bubble and "above" bubble
		createReq := ConstructRequest{
			Type:        "create_construct",
			ConstructID: fmt.Sprintf("bubble_group_%d", bubble.Index),
			Parts: []Part{
				// Part 1: Sphere AT bubble position
				{
					ID: "on_bubble", Type: "sphere", Size: Vector3{1.5, 1.5, 1.5},
					Pos: bubblePos, Color: Vector3{0, 1, 0}, Locked: true,
				},
				// Part 2: Sphere 5m ABOVE bubble position
				{
					ID: "above_bubble", Type: "sphere", Size: Vector3{1.5, 1.5, 1.5},
					Pos: Vector3{
						bubblePos[0] + up[0]*5.0,
						bubblePos[1] + up[1]*5.0,
						bubblePos[2] + up[2]*5.0,
					},
					Color: Vector3{1, 0.5, 0}, Locked: true,
				},
				// Part 3: A connector beam
				{
					ID: "connector", Type: "capsule", Size: Vector3{0.3, 5, 0},
					Pos: Vector3{
						bubblePos[0] + up[0]*2.5,
						bubblePos[1] + up[1]*2.5,
						bubblePos[2] + up[2]*2.5,
					},
					Color: Vector3{0.5, 0.5, 0.5}, Locked: true,
				},
			},
		}

		cData, _ := json.Marshal(createReq)
		writePacket(conn, cData)

		// Small delay to not overwhelm the TCP buffer/server
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Printf("✅ Spawning complete for %d bubble groups.\n", len(state.Bubbles))
	time.Sleep(1 * time.Second)
}
