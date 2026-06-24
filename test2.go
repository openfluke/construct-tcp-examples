package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"sync"
	"time"
)

// --- Generators ---

func StartSnakeBot(wg *sync.WaitGroup, id int, offset Vector3) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ [SnakeBot-%d] Failed to connect: %v\n", id, err)
		return
	}
	defer conn.Close()

	constructID := fmt.Sprintf("snake_bot_%d", id)
	fmt.Printf("🐍 [SnakeBot-%d] Connected. Creating construct...\n", id)

	parts := []Part{}
	joints := []Joint{}
	segments := 8

	// Head
	parts = append(parts, Part{
		ID: "head", Type: "sphere", Size: Vector3{0.4, 0.4, 0.4},
		Pos:   Vector3{offset[0], offset[1] + 2, offset[2]},
		Color: Vector3{0.2, 0.8, 0.2}, Groups: []string{"lasso_target"},
	})

	for i := 0; i < segments; i++ {
		z := offset[2] - float32(i+1)*0.7
		pid := fmt.Sprintf("seg_%d", i)

		parts = append(parts, Part{
			ID: pid, Type: "box", Size: Vector3{0.5, 0.5, 0.5},
			Pos:   Vector3{offset[0], offset[1] + 2, z},
			Color: Vector3{0.2, float32(i) * 0.1, 0.2}, Groups: []string{"lasso_target"},
		})

		prev := "head"
		if i > 0 {
			prev = fmt.Sprintf("seg_%d", i-1)
		}

		joints = append(joints, Joint{
			Type: "pin", A: prev, B: pid,
			Pos: Vector3{offset[0], offset[1] + 2, z + 0.35},
		})
	}

	req := ConstructRequest{Type: "create_construct", ConstructID: constructID, Parts: parts, Joints: joints}
	data, _ := json.Marshal(req)
	writePacket(conn, data)

	// Animate
	fmt.Printf("🐍 [SnakeBot-%d] Slithering...\n", id)
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	for range ticker.C {
		elapsed := float32(time.Since(startTime).Seconds())
		if elapsed > 60 {
			break
		}

		updates := []PartUpdate{}
		// Make head bob
		torque := Vector3{float32(math.Sin(float64(elapsed*3))) * 10, 0, 0}
		updates = append(updates, PartUpdate{PartID: "head", Torque: &torque})

		// Wave segments
		for i := 0; i < segments; i++ {
			pid := fmt.Sprintf("seg_%d", i)
			wave := float32(math.Sin(float64(elapsed*5+float32(i)))) * 15
			t := Vector3{0, wave, 0}
			updates = append(updates, PartUpdate{PartID: pid, Torque: &t})
		}

		uReq := UpdateRequest{Type: "update_construct", ConstructID: constructID, Updates: updates}
		uData, _ := json.Marshal(uReq)
		writePacket(conn, uData)
	}
}

func StartBoxTower(wg *sync.WaitGroup, id int, offset Vector3) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ [BoxTower-%d] Failed to connect: %v\n", id, err)
		return
	}
	defer conn.Close()

	constructID := fmt.Sprintf("box_tower_%d", id)
	fmt.Printf("🗼 [BoxTower-%d] Building tower...\n", id)

	parts := []Part{}
	joints := []Joint{} // No joints, just stacking physics objects (if we set them not locked)

	// Base (Locked)
	parts = append(parts, Part{
		ID: "base", Type: "box", Size: Vector3{2, 0.5, 2},
		Pos:    Vector3{offset[0], offset[1], offset[2]},
		Locked: true, Color: Vector3{0.5, 0.5, 0.5}, Groups: []string{"lasso_target"},
	})

	height := 6
	for i := 0; i < height; i++ {
		pid := fmt.Sprintf("block_%d", i)
		parts = append(parts, Part{
			ID: pid, Type: "box", Size: Vector3{0.8, 0.8, 0.8},
			Pos:   Vector3{offset[0], offset[1] + 1 + float32(i)*0.9, offset[2]},
			Color: Vector3{1, 0.5, 0}, Groups: []string{"lasso_target"},
		})
	}

	req := ConstructRequest{Type: "create_construct", ConstructID: constructID, Parts: parts, Joints: joints}
	data, _ := json.Marshal(req)
	writePacket(conn, data)

	// Animate: Add occasional random force to wobble
	fmt.Printf("🗼 [BoxTower-%d] Wobbling...\n", id)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	for range ticker.C {
		elapsed := float32(time.Since(startTime).Seconds())
		if elapsed > 60 {
			break
		}

		updates := []PartUpdate{}
		// Push middle block
		mid := fmt.Sprintf("block_%d", height/2)
		force := Vector3{float32(math.Sin(float64(elapsed))) * 20, 0, 0}
		updates = append(updates, PartUpdate{PartID: mid, Torque: &force})

		uReq := UpdateRequest{Type: "update_construct", ConstructID: constructID, Updates: updates}
		uData, _ := json.Marshal(uReq)
		writePacket(conn, uData)
	}
}

func StartJellyfish(wg *sync.WaitGroup, id int, offset Vector3) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ [Jellyfish-%d] Failed to connect: %v\n", id, err)
		return
	}
	defer conn.Close()

	constructID := fmt.Sprintf("jellyfish_%d", id)
	fmt.Printf("🪼 [Jellyfish-%d] Floating...\n", id)

	parts := []Part{}
	joints := []Joint{}

	// Bell
	parts = append(parts, Part{
		ID: "bell", Type: "sphere", Size: Vector3{1.5, 1.5, 1.5}, // Radius
		Pos:   Vector3{offset[0], offset[1] + 5, offset[2]},
		Color: Vector3{0.8, 0.4, 0.9}, Groups: []string{"lasso_target"},
		Locked: true, // Float in air
	})

	tentacles := 6
	segPerTentacle := 4

	for t := 0; t < tentacles; t++ {
		angle := float64(t) * (2 * math.Pi / float64(tentacles))
		xOff := float32(math.Cos(angle)) * 0.8
		zOff := float32(math.Sin(angle)) * 0.8

		parentID := "bell"

		for s := 0; s < segPerTentacle; s++ {
			pid := fmt.Sprintf("t%d_s%d", t, s)
			yPos := (offset[1] + 5) - 1.2 - float32(s)*0.6

			parts = append(parts, Part{
				ID: pid, Type: "capsule", Size: Vector3{0.1, 0.5, 0},
				Pos:   Vector3{offset[0] + xOff, yPos, offset[2] + zOff},
				Color: Vector3{0.6, 0.2, 0.7}, Groups: []string{"lasso_target"},
			})

			joints = append(joints, Joint{
				Type: "pin", A: parentID, B: pid,
				Pos: Vector3{offset[0] + xOff, yPos + 0.3, offset[2] + zOff},
			})
			parentID = pid
		}
	}

	req := ConstructRequest{Type: "create_construct", ConstructID: constructID, Parts: parts, Joints: joints}
	data, _ := json.Marshal(req)
	writePacket(conn, data)

	// Animate tentacles
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	for range ticker.C {
		elapsed := float32(time.Since(startTime).Seconds())
		if elapsed > 60 {
			break
		}

		updates := []PartUpdate{}

		// Pulse tentacles
		for t := 0; t < tentacles; t++ {
			for s := 0; s < segPerTentacle; s++ {
				pid := fmt.Sprintf("t%d_s%d", t, s)

				// Expand/Contract motion
				factor := float32(math.Sin(float64(elapsed*2+float32(s)*0.5))) * 5.0

				// Radial outward force
				angle := float64(t) * (2 * math.Pi / float64(tentacles))
				force := Vector3{
					float32(math.Cos(angle)) * factor,
					0,
					float32(math.Sin(angle)) * factor,
				}

				updates = append(updates, PartUpdate{PartID: pid, Torque: &force})
			}
		}

		uReq := UpdateRequest{Type: "update_construct", ConstructID: constructID, Updates: updates}
		uData, _ := json.Marshal(uReq)
		writePacket(conn, uData)
	}
}

func RunTest2() {
	var wg sync.WaitGroup

	// TCP dev sandbox near origin (not multiverse cell indices).
	baseX := float32(0)
	baseY := float32(4)
	baseZ := float32(0)

	fmt.Println("🚀 Starting Multi-Construct Load Test...")

	// Spawn SnakeBot
	wg.Add(1)
	go StartSnakeBot(&wg, 1, Vector3{baseX + 5, baseY, baseZ + 5})

	// Spawn BoxTower
	wg.Add(1)
	go StartBoxTower(&wg, 1, Vector3{baseX - 5, baseY, baseZ + 5})

	// Spawn Jellyfish
	wg.Add(1)
	go StartJellyfish(&wg, 1, Vector3{baseX, baseY + 5, baseZ - 5})

	// Spawn Another Snake
	wg.Add(1)
	go StartSnakeBot(&wg, 2, Vector3{baseX + 10, baseY, baseZ})

	wg.Wait()
	fmt.Println("✅ All constructs finished simulation.")
}
