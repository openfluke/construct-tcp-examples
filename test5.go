package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"sync"
	"time"
)

func RunTest5() {
	fmt.Println("🛰️  Initializing Auto-Discovery Satellite System...")

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
	buf := make([]byte, 8192) // Increased buffer for state
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

	fmt.Printf("📍 Player at: %.1f, %.1f, %.1f\n", state.PlayerPos[0], state.PlayerPos[1], state.PlayerPos[2])
	fmt.Printf("🪐 Planet Center: %.1f, %.1f, %.1f (Radius: %.1f)\n", state.PlanetCenter[0], state.PlanetCenter[1], state.PlanetCenter[2], state.PlanetRadius)

	// 2. Spawn Satellites
	var wg sync.WaitGroup
	pCenter := Vector3{state.PlanetCenter[0], state.PlanetCenter[1], state.PlanetCenter[2]}
	pPos := Vector3{state.PlayerPos[0], state.PlayerPos[1], state.PlayerPos[2]}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go SpawnSatellite(&wg, i, pCenter, pPos, state.PlanetRadius)
	}

	wg.Wait()
}

func SpawnSatellite(wg *sync.WaitGroup, id int, center Vector3, playerPos Vector3, radius float32) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		return
	}
	defer conn.Close()

	constructID := fmt.Sprintf("sat_%d", id)
	orbitDist := radius + 50.0 + float32(id)*10.0

	// Start position
	angle := float64(id) * 1.2
	pos := Vector3{
		center[0] + float32(math.Cos(angle))*orbitDist,
		center[1] + float32(math.Sin(angle*0.5))*orbitDist,
		center[2] + float32(math.Sin(angle))*orbitDist,
	}

	parts := []Part{
		{ID: "core", Type: "box", Size: Vector3{2, 2, 2}, Pos: pos, Color: Vector3{0.8, 0.8, 0.9}, Locked: true},
		{ID: "panel_l", Type: "box", Size: Vector3{4, 0.2, 3}, Pos: Vector3{pos[0] - 3, pos[1], pos[2]}, Color: Vector3{0, 0.4, 0.8}, Locked: true},
		{ID: "panel_r", Type: "box", Size: Vector3{4, 0.2, 3}, Pos: Vector3{pos[0] + 3, pos[1], pos[2]}, Color: Vector3{0, 0.4, 0.8}, Locked: true},
	}

	req := ConstructRequest{Type: "create_construct", ConstructID: constructID, Parts: parts}
	data, _ := json.Marshal(req)
	writePacket(conn, data)

	// Animate Orbit
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()
	startTime := time.Now()

	for range ticker.C {
		if time.Since(startTime).Seconds() > 120 {
			break
		}

		t := time.Since(startTime).Seconds() * 0.2
		curAngle := angle + t

		newPos := Vector3{
			center[0] + float32(math.Cos(curAngle))*orbitDist,
			center[1] + float32(math.Sin(curAngle*0.5))*orbitDist,
			center[2] + float32(math.Sin(curAngle))*orbitDist,
		}

		update := UpdateRequest{
			Type:        "update_construct",
			ConstructID: constructID,
			Updates: []PartUpdate{
				{PartID: "core", Position: &newPos},
				{PartID: "panel_l", Position: &Vector3{newPos[0] - 3, newPos[1], newPos[2]}},
				{PartID: "panel_r", Position: &Vector3{newPos[0] + 3, newPos[1], newPos[2]}},
			},
		}

		uData, _ := json.Marshal(update)
		writePacket(conn, uData)
	}
}
