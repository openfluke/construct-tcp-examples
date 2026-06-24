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

// --- Procedural Building Generator ---

func GenerateBuildingName(bType int, height int) string {
	types := []string{"Skyscraper", "Helix Tower", "Pyramid", "Fortress", "Modern House"}
	adjectives := []string{"Grand", "Obsidian", "Crystal", "Forgotten", "Neo", "Cyber", "Mega", "Quantum"}

	adj := adjectives[rand.Intn(len(adjectives))]
	typ := types[bType]

	return fmt.Sprintf("%s %s (H:%d)", adj, typ, height)
}

// --- Planetary Logic ---

func StartBuildingOnPlanet(wg *sync.WaitGroup, id int, planetCenter Vector3, surfacePos Vector3, normal Vector3) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		return
	}
	defer conn.Close()

	// Building Basis (Local Coordinates -> Global)
	bX, bY, bZ := MakeBasis(normal)

	// Calculate global orientation for the building
	// YXZ ordering (Godot standard)
	var ex, ey, ez float64
	m00, _, m02 := float64(bX[0]), float64(bY[0]), float64(bZ[0])
	m10, m11, m12 := float64(bX[1]), float64(bY[1]), float64(bZ[1])
	m20, _, m22 := float64(bX[2]), float64(bY[2]), float64(bZ[2])

	if m12 > 1.0 {
		m12 = 1.0
	}
	if m12 < -1.0 {
		m12 = -1.0
	}
	ex = math.Asin(-m12)

	if math.Abs(m12) < 0.99999 {
		ey = math.Atan2(m02, m22)
		ez = math.Atan2(m10, m11)
	} else {
		ey = math.Atan2(-m20, m00)
		ez = 0
	}
	buildingRot := Vector3{RadToDeg(ex), RadToDeg(ey), RadToDeg(ez)}

	// Building DNA
	bType := rand.Intn(5)
	height := 5 + rand.Intn(15)
	baseScale := 1.0 + rand.Float32()*0.5

	primaryCol := RandomColor()
	secondaryCol := RandomColor()

	bName := GenerateBuildingName(bType, height)
	constructID := sanitizeID(fmt.Sprintf("bldg_%d_%s", id, bName))

	fmt.Printf("🏗️ Constructing: '%s' (H:%d)\n", bName, height)

	parts := []Part{}
	addBox := func(idSuffix string, w, h, d float32, lx, ly, lz float32, col Vector3) {
		localVec := Vector3{lx, ly, lz}
		globalPos := TransformPoint(surfacePos, bX, bY, bZ, localVec)
		parts = append(parts, Part{
			ID: idSuffix, Type: "box", Size: Vector3{w, h, d},
			Pos: globalPos, Color: col, Locked: false,
			Rot:    buildingRot, // Align with planet surface!
			Groups: []string{"lasso_target"},
		})
	}

	// Logic for different building types
	switch bType {
	case 0: // Skyscraper
		addBox("base", 2*baseScale, 1*baseScale, 2*baseScale, 0, 0.5*baseScale, 0, primaryCol)
		for i := 1; i < height; i++ {
			addBox(fmt.Sprintf("f_%d", i), 1.8*baseScale, 1*baseScale, 1.8*baseScale, 0, 0.5*baseScale+float32(i)*baseScale, 0, secondaryCol)
		}
	case 1: // Helix
		addBox("base", 2*baseScale, 1*baseScale, 2*baseScale, 0, 0.5*baseScale, 0, primaryCol)
		for i := 1; i < height; i++ {
			angle := float64(i) * 0.4
			lx := float32(math.Cos(angle)) * 0.5 * baseScale
			lz := float32(math.Sin(angle)) * 0.5 * baseScale
			addBox(fmt.Sprintf("f_%d", i), 1.2*baseScale, 0.8*baseScale, 1.2*baseScale, lx, 0.5*baseScale+float32(i)*0.8*baseScale, lz, secondaryCol)
		}
	case 2: // Pyramid
		for i := 0; i < height; i++ {
			w := (float32(height) - float32(i)) * 0.5 * baseScale
			addBox(fmt.Sprintf("f_%d", i), w, 0.8*baseScale, w, 0, 0.4*baseScale+float32(i)*0.8*baseScale, 0, primaryCol)
		}
	default: // Random blocks
		for i := 0; i < height; i++ {
			lx := (rand.Float32() - 0.5) * 2 * baseScale
			lz := (rand.Float32() - 0.5) * 2 * baseScale
			addBox(fmt.Sprintf("f_%d", i), 1.5, 1.5, 1.5, lx, 0.75+float32(i)*1.5, lz, secondaryCol)
		}
	}

	req := ConstructRequest{Type: "create_construct", ConstructID: constructID, Parts: parts}
	d, _ := json.Marshal(req)
	writePacket(conn, d)

	// Heartbeat/Keep-alive (not strictly needed but good for long tests)
	ticker := time.NewTicker(2000 * time.Millisecond)
	defer ticker.Stop()
	st := time.Now()
	for range ticker.C {
		if time.Since(st).Seconds() > 120 {
			break
		}
		// Send a dummy heartbeat to keep the client session active if server tracks it
		// (Currently we just keep conn open)
	}
}

func RunTest4() {
	var wg sync.WaitGroup
	fmt.Println("🌆 Starting Test 4: PLANETARY ARCHITECTURE 🌆")

	// TCP dev sandbox: keep builds near the local floor; planetCenter is a reference
	// point for surface normals (synthetic "planet" at origin scale).
	origin := Vector3{0, 4, 0}
	planetCenter := Vector3{0, 0, 0}

	for i := 0; i < 30; i++ {
		wg.Add(1)

		// Distribute around the origin on the sphere
		theta := float64(i) * 0.3
		dist := float32(10 + rand.Float32()*20)

		surfacePos := Vector3{
			origin[0] + float32(math.Cos(theta))*dist,
			origin[1],
			origin[2] + float32(math.Sin(theta))*dist,
		}

		// Calculate normal relative to planet center
		vecToSurf := Vector3{surfacePos[0] - planetCenter[0], surfacePos[1] - planetCenter[1], surfacePos[2] - planetCenter[2]}
		normal := Normalize(vecToSurf)

		go StartBuildingOnPlanet(&wg, i, planetCenter, surfacePos, normal)
		time.Sleep(50 * time.Millisecond)
	}

	wg.Wait()
	fmt.Println("✅ City construction finished.")
}
