package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"
)

// Config
const (
	ServerAddress = "127.0.0.1:17000"
)

func RunTest6() {
	fmt.Println("🚀 Starting PrimeCraft Universal Query Test (test6)")
	conn, err := net.Dial("tcp", ServerAddress)
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Println("✅ Connected to ConstructServer")

	// Helper to send JSON
	send := func(data map[string]interface{}) {
		msg, _ := json.Marshal(data)
		writePacket(conn, msg)
	}

	// Handle interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println("\n🛑 Disconnecting...")
		conn.Close()
		os.Exit(0)
	}()

	// Query Loop
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Buffer for reading
	buf := make([]byte, 65536) // Large buffer for lists

	for range ticker.C {
		fmt.Println("\n📡 Querying World State...")

		// 1. Query Planets
		send(map[string]interface{}{"type": "query_nearby_planets"})
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("❌ Read Error: %v\n", err)
			break
		}

		// Parse Planets
		var planetResp NearbyPlanetsResponse
		inputData := buf[:n]

		// The server RESPONSE is not currently length-prefixed in C#,
		// only the INCOMING stream is buffered.
		// This test client assumes a single synchronous read captures the response.
		if err := json.Unmarshal(inputData, &planetResp); err == nil && planetResp.Type == "nearby_planets_response" {
			fmt.Printf("🪐 Nearby Planets (%d):\n", planetResp.Count)
			for _, p := range planetResp.Planets {
				fmt.Printf("   - %s (Seed: %s) @ [%.1f, %.1f, %.1f] R=%.1f\n",
					p.Name, p.Seed, p.Position[0], p.Position[1], p.Position[2], p.Radius)
			}
		} else {
			fmt.Printf("⚠️ Received unexpected data for planets: %s (Err: %v)\n", string(inputData), err)
		}

		// 2. Query Players
		send(map[string]interface{}{"type": "query_players"})
		n, err = conn.Read(buf)
		if err != nil {
			fmt.Printf("❌ Read Error (Players): %v\n", err)
			break
		}

		inputData = buf[:n]
		var playersResp PlayersResponse
		if err := json.Unmarshal(inputData, &playersResp); err == nil && playersResp.Type == "players_response" {
			fmt.Printf("👤 Players (%d):\n", playersResp.Count)
			for _, p := range playersResp.Players {
				userStr := "AI"
				if p.IsUser {
					userStr = "USER"
				}
				fmt.Printf("   - [%s] %s (Index: %d) @ [%.1f, %.1f, %.1f]\n",
					userStr, p.Name, p.Index, p.Position[0], p.Position[1], p.Position[2])
			}
		} else {
			fmt.Printf("⚠️ Received unexpected data for players: %s (Err: %v)\n", string(inputData), err)
		}
	}
}
