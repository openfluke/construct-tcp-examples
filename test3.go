package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sync"
	"time"
)

// --- Procedural Generation Utils ---

func GenerateName(planType int, color Vector3) string {
	adjectives := []string{"Spiked", "Smooth", "Giant", "Tiny", "Swift", "Lumbering", "Glowing", "Shadow", "Ancient", "Neon"}
	colors := []string{"Red", "Green", "Blue", "Golden", "Purple", "Cyan", "Orange", "Crimson", "Emerald", "Obsidian"}
	nouns := []string{"Worm", "Star", "Walker", "Cloud", "Totem"}

	// Pick color name based on dominant channel
	cIdx := 0
	if color[0] > color[1] && color[0] > color[2] {
		cIdx = 0
		if color[0] > 0.8 {
			cIdx = 7
		}
	} // Red/Crimson
	if color[1] > color[0] && color[1] > color[2] {
		cIdx = 1
		if color[1] > 0.8 {
			cIdx = 8
		}
	} // Green/Emerald
	if color[2] > color[0] && color[2] > color[1] {
		cIdx = 2
		if color[2] > 0.8 {
			cIdx = 5
		}
	} // Blue/Cyan

	adj := adjectives[rand.Intn(len(adjectives))]
	colName := colors[cIdx]
	if rand.Float32() > 0.7 {
		colName = colors[rand.Intn(len(colors))]
	} // Random chance for "Golden" etc

	return fmt.Sprintf("%s %s %s", adj, colName, nouns[planType])
}

// --- Generator ---

func StartUniqueCreature(wg *sync.WaitGroup, id int, offset Vector3) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ [ID-%d] Connect Fail: %v\n", id, err)
		return
	}
	defer conn.Close()

	// --- 1. DNA Generation ---
	planType := rand.Intn(5) // 0=Worm, 1=Star, 2=Walker, 3=Cloud, 4=Totem

	// Phenotypes
	baseColor := RandomColor()
	accentColor := RandomColor()
	scale := 0.6 + rand.Float32()*1.4 // Size variation (0.6x to 2.0x)
	mutation := rand.Float32()        // 0.0-1.0 gene for shape variation

	creatureName := GenerateName(planType, baseColor)
	constructID := sanitizeID(fmt.Sprintf("creature_%d_%s", id, creatureName))

	fmt.Printf("🧬 Spawned: '%s' (Type: %d, Scale: %.1fx)\n", creatureName, planType, scale)

	parts := []Part{}
	joints := []Joint{}

	// --- 2. Morphology Generation (Building the Body) ---
	switch planType {
	case 0: // WORM Class
		// Variations: Short/Fat, Long/Thin, Tapered
		segments := 4 + rand.Intn(12)
		thickness := (0.3 + rand.Float32()*0.5) * scale

		for i := 0; i < segments; i++ {
			pid := fmt.Sprintf("seg_%d", i)
			z := offset[2] - float32(i)*(thickness*1.8)

			// Tapering logic
			mySize := thickness
			if mutation > 0.5 {
				mySize *= (1.0 - float32(i)/float32(segments)*0.6)
			} // Tail taper

			col := baseColor
			if i%2 == 0 && mutation < 0.3 {
				col = accentColor
			} // Striped mutation

			pType := "sphere"
			if mutation > 0.7 {
				pType = "box"
			} // Blocky worm mutation

			parts = append(parts, Part{
				ID: pid, Type: pType, Size: Vector3{mySize, mySize, mySize},
				Pos:   Vector3{offset[0], offset[1] + 2, z},
				Color: col, Groups: []string{"lasso_target"},
			})

			if i > 0 {
				prev := fmt.Sprintf("seg_%d", i-1)
				joints = append(joints, Joint{
					Type: "pin", A: prev, B: pid,
					Pos: Vector3{offset[0], offset[1] + 2, z + thickness},
				})
			}
		}

	case 1: // STAR Class
		arms := 3 + rand.Intn(5)
		thickness := 0.4 * scale
		parts = append(parts, Part{
			ID: "core", Type: "sphere", Size: Vector3{thickness * 1.5, thickness * 1.5, thickness * 1.5},
			Pos:   Vector3{offset[0], offset[1] + 2, offset[2]},
			Color: accentColor, Groups: []string{"lasso_target"},
		})

		for i := 0; i < arms; i++ {
			angle := (float64(i) * 2 * math.Pi / float64(arms)) + float64(mutation)
			length := (1.5 + rand.Float32()*1.5) * scale

			pid := fmt.Sprintf("arm_%d", i)
			pX := offset[0] + float32(math.Cos(angle))*length
			pZ := offset[2] + float32(math.Sin(angle))*length

			parts = append(parts, Part{
				ID: pid, Type: "capsule", Size: Vector3{thickness * 0.5, length, 0},
				Pos:   Vector3{pX, offset[1] + 2, pZ},
				Color: baseColor, Groups: []string{"lasso_target"},
				Rot: Vector3{0, float32(angle * 180 / math.Pi), 90},
			})

			joints = append(joints, Joint{
				Type: "pin", A: "core", B: pid,
				Pos: Vector3{offset[0] + float32(math.Cos(angle))*0.3, offset[1] + 2, offset[2] + float32(math.Sin(angle))*0.3},
			})
		}

	case 2: // WALKER Class
		bodySize := scale
		parts = append(parts, Part{
			ID: "body", Type: "box", Size: Vector3{bodySize, bodySize * 0.6, bodySize * 1.2},
			Pos:   Vector3{offset[0], offset[1] + 2 + bodySize, offset[2]},
			Color: baseColor, Groups: []string{"lasso_target"},
		})

		legs := 4
		if mutation > 0.8 {
			legs = 6
		}

		for i := 0; i < legs; i++ {
			side := float32(1.0)
			if i%2 == 1 {
				side = -1.0
			}
			zPos := (float32(i/2) - float32(legs)/4.0 + 0.5) * scale

			lID := fmt.Sprintf("leg_%d", i)
			parts = append(parts, Part{
				ID: lID, Type: "capsule", Size: Vector3{0.15 * scale, 1.2 * scale, 0},
				Pos:   Vector3{offset[0] + side*bodySize, offset[1] + 1, offset[2] + zPos},
				Color: accentColor, Groups: []string{"lasso_target"},
			})

			joints = append(joints, Joint{
				Type: "pin", A: "body", B: lID,
				Pos: Vector3{offset[0] + side*bodySize*0.5, offset[1] + 2 + bodySize, offset[2] + zPos},
			})
		}

	case 3: // CLOUD (Multi-Sphere)
		blobs := 5 + rand.Intn(10)
		for i := 0; i < blobs; i++ {
			pid := fmt.Sprintf("b_%d", i)
			bSize := (0.5 + rand.Float32()*1.0) * scale
			lX := (rand.Float32() - 0.5) * 2 * scale
			lY := (rand.Float32() - 0.5) * 2 * scale
			lZ := (rand.Float32() - 0.5) * 2 * scale

			parts = append(parts, Part{
				ID: pid, Type: "sphere", Size: Vector3{bSize, bSize, bSize},
				Pos:   Vector3{offset[0] + lX, offset[1] + 5 + lY, offset[2] + lZ},
				Color: baseColor, Locked: true, Groups: []string{"lasso_target"},
			})
		}

	case 4: // TOTEM (Stacked)
		blocks := 3 + rand.Intn(5)
		for i := 0; i < blocks; i++ {
			pid := fmt.Sprintf("t_%d", i)
			tSize := (0.8 + rand.Float32()) * scale
			parts = append(parts, Part{
				ID: pid, Type: "box", Size: Vector3{tSize, tSize, tSize},
				Pos:   Vector3{offset[0], offset[1] + float32(i)*tSize, offset[2]},
				Color: baseColor, Locked: true, Groups: []string{"lasso_target"},
				Rot: Vector3{0, float32(i) * 15, 0},
			})
		}
	}

	// --- 3. Finalize and Send ---
	req := ConstructRequest{Type: "create_construct", ConstructID: constructID, Parts: parts, Joints: joints}
	data, _ := json.Marshal(req)
	writePacket(conn, data)

	// --- 4. Behavior Loop ---
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	phase := 0.0

	for range ticker.C {
		if time.Since(startTime).Seconds() > 120 {
			break
		}
		phase += 0.1

		updates := []PartUpdate{}

		switch planType {
		case 0: // Worm slither
			for i := 0; i < len(parts); i++ {
				force := Vector3{0, float32(math.Sin(phase+float64(i)*0.4) * 20), 0}
				updates = append(updates, PartUpdate{PartID: parts[i].ID, Torque: &force})
			}
		case 1: // Star spinning
			for i := 1; i < len(parts); i++ {
				force := Vector3{0, 15, 0}
				updates = append(updates, PartUpdate{PartID: parts[i].ID, Torque: &force})
			}
		case 2: // Walker leg cycle
			for i := 0; i < len(joints); i++ {
				side := float32(1.0)
				if i%2 == 1 {
					side = -1.0
				}
				force := Vector3{0, 0, float32(math.Sin(phase+float64(i)*math.Pi)) * 40 * side}
				updates = append(updates, PartUpdate{PartID: fmt.Sprintf("leg_%d", i), Torque: &force})
			}
		case 3: // Cloud undulation
			for i := 0; i < len(parts); i++ {
				y := float32(math.Sin(phase+float64(i)*0.5)) * 0.05
				newPos := parts[i].Pos
				newPos[1] += y
				updates = append(updates, PartUpdate{PartID: parts[i].ID, Position: &newPos})
			}
		}

		if len(updates) > 0 {
			uReq := UpdateRequest{Type: "update_construct", ConstructID: constructID, Updates: updates}
			uData, _ := json.Marshal(uReq)
			writePacket(conn, uData)
		}
	}
}

func RunTest3() {
	var wg sync.WaitGroup
	fmt.Println("🌟 Starting Test 3: PROCEDURAL BESTIARY 🌟")

	// Spawn a variety of creatures
	for i := 0; i < 20; i++ {
		wg.Add(1)
		spawnX := float32((rand.Float32() - 0.5) * 40)
		spawnZ := float32((rand.Float32() - 0.5) * 40)
		go StartUniqueCreature(&wg, i, Vector3{spawnX, 4, spawnZ})

		time.Sleep(100 * time.Millisecond) // Stagger spawns
	}

	wg.Wait()
	fmt.Println("✅ All creatures have evolved and finished.")
}
