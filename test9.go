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

// --- Extended Controllers ---

// 1. Magical Tree
type MagicTreeController struct {
	ConstructID string
	LeafIDs     []string
	Phase       float64
}

func (c *MagicTreeController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id
	b := GetBasis(normal)
	rot := GetEuler(b)
	surfacePos := Add(center, Mul(normal, radius))

	trunkHeight := 4.0 + rand.Float32()*4.0
	trunkColor := Vector3{0.2, 0.15, 0.3} // Dark mystical wood
	leafColor := Vector3{0.1, 0.8, 0.4}   // Glowing green
	if rand.Float64() > 0.5 {
		leafColor = Vector3{0.8, 0.2, 0.8}
	} // Or purple

	parts := []Part{
		{
			ID: "trunk", Type: "capsule", Size: Vector3{0.5, trunkHeight, 0},
			Pos:   TransformPoint(surfacePos, b.X, b.Y, b.Z, Vector3{0, trunkHeight * 0.5, 0}),
			Color: trunkColor, Locked: true, Rot: rot,
		},
	}

	for i := 0; i < 5; i++ {
		pid := fmt.Sprintf("leaf_%d", i)
		c.LeafIDs = append(c.LeafIDs, pid)
		lSize := 1.5 + rand.Float32()*1.0
		offset := Vector3{
			(rand.Float32() - 0.5) * 3,
			trunkHeight + rand.Float32()*2,
			(rand.Float32() - 0.5) * 3,
		}
		parts = append(parts, Part{
			ID: pid, Type: "sphere", Size: Vector3{lSize, lSize, lSize},
			Pos:   TransformPoint(surfacePos, b.X, b.Y, b.Z, offset),
			Color: leafColor, Locked: true,
		})
	}

	return ConstructRequestWithJoints{Type: "create_construct", ConstructID: id, Parts: parts}, nil
}

func (c *MagicTreeController) Tick(dt float64, conn net.Conn) {
	c.Phase += dt * 2.0
	// Make leaves gently bob
}

// 2. Mystical Skyscraper
type MagicBuildingController struct {
	ConstructID string
}

func (c *MagicBuildingController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id
	b := GetBasis(normal)
	rot := GetEuler(b)
	surfacePos := Add(center, Mul(normal, radius))

	height := 10 + rand.Intn(20)
	pCol := Vector3{0.1, 0.1, 0.2} // Dark base
	sCol := Vector3{0, 0.8, 1.0}   // Neon windows/trim

	parts := []Part{}
	for i := 0; i < height; i++ {
		y := float32(i) * 1.5
		w := 3.0 - float32(i)*0.1
		if w < 1.0 {
			w = 1.0
		}

		col := pCol
		if i%3 == 0 {
			col = sCol
		}

		pid := fmt.Sprintf("floor_%d", i)
		parts = append(parts, Part{
			ID: pid, Type: "box", Size: Vector3{w, 1.4, w},
			Pos:   TransformPoint(surfacePos, b.X, b.Y, b.Z, Vector3{0, y + 0.7, 0}),
			Color: col, Locked: true, Rot: rot,
		})
	}

	return ConstructRequestWithJoints{Type: "create_construct", ConstructID: id, Parts: parts}, nil
}

func (c *MagicBuildingController) Tick(dt float64, conn net.Conn) {}

// 3. Mana Worm (Animated Creature)
type ManaWormController struct {
	ConstructID string
	BodyParts   []string
	Phase       float64
	BasePos     Vector3
	Normal      Basis
}

func (c *ManaWormController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id
	c.Normal = GetBasis(normal)
	c.BasePos = Add(center, Mul(normal, radius+1.0))

	segments := 6 + rand.Intn(6)
	col := Vector3{rand.Float32(), 1.0, 0.5}

	parts := []Part{}
	joints := []JointDef{}

	for i := 0; i < segments; i++ {
		pid := fmt.Sprintf("seg_%d", i)
		c.BodyParts = append(c.BodyParts, pid)

		size := 0.8 - float32(i)*0.05
		lPos := Vector3{0, 0, float32(i) * 1.2}

		parts = append(parts, Part{
			ID: pid, Type: "sphere", Size: Vector3{size, size, size},
			Pos:   TransformPoint(c.BasePos, c.Normal.X, c.Normal.Y, c.Normal.Z, lPos),
			Color: col, Groups: []string{"magic_animal"},
		})

		if i > 0 {
			prev := fmt.Sprintf("seg_%d", i-1)
			jPos := TransformPoint(c.BasePos, c.Normal.X, c.Normal.Y, c.Normal.Z, Vector3{0, 0, float32(i)*1.2 - 0.6})
			joints = append(joints, JointDef{Type: "pin", A: prev, B: pid, Pos: []float32{jPos[0], jPos[1], jPos[2]}})
		}
	}

	return ConstructRequestWithJoints{Type: "create_construct", ConstructID: id, Parts: parts, Joints: joints}, nil
}

func (c *ManaWormController) Tick(dt float64, conn net.Conn) {
	c.Phase += dt * 5.0
	updates := []map[string]interface{}{}
	for i, pid := range c.BodyParts {
		f := float32(math.Sin(c.Phase+float64(i)*0.8)) * 20.0
		torque := []float32{0, f, 0}
		updates = append(updates, map[string]interface{}{"part_id": pid, "torque": torque})
	}
	msg, _ := json.Marshal(map[string]interface{}{"type": "update_construct", "construct_id": c.ConstructID, "updates": updates})
	writePacket(conn, msg)
}

// 4. Floating Crystal (Glowing Locked Decoration)
type CrystalController struct {
	ConstructID string
	BasePos     Vector3
	Color       Vector3
}

func (c *CrystalController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id
	c.Color = Vector3{0.5 + rand.Float32()*0.5, 0.2 + rand.Float32()*0.3, 0.8 + rand.Float32()*0.2} // Purple/Pink/Blue

	height := radius + 2.0 + rand.Float32()*5.0
	c.BasePos = VecAdd(center, VecMul(normal, height))

	parts := []Part{
		{
			ID: "crystal_core", Type: "box", Size: Vector3{0.5, 2.5, 0.5},
			Pos: c.BasePos, Color: c.Color, Locked: true, Groups: []string{"magic_source"},
		},
		{
			ID: "glow_1", Type: "sphere", Size: Vector3{1.2, 1.2, 1.2},
			Pos: c.BasePos, Color: c.Color, Locked: true,
		},
	}

	return ConstructRequestWithJoints{
		Type:        "create_construct",
		ConstructID: id,
		Parts:       parts,
	}, nil
}

func (c *CrystalController) Tick(dt float64, conn net.Conn) {}

// 5. Orbiting Wisps (A central point with tiny floating bits)
type WispController struct {
	ConstructID string
	Center      Vector3
	Phase       float64
	Wisps       []string
}

func (c *WispController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id
	c.Phase = rand.Float64() * 10

	c.Center = VecAdd(center, VecMul(normal, radius+5.0+rand.Float32()*10.0))

	parts := []Part{
		{ID: "anchor", Type: "sphere", Size: Vector3{0.1, 0.1, 0.1}, Pos: c.Center, Color: Vector3{1, 1, 1}, Locked: true},
	}

	wispCount := 4
	for i := 0; i < wispCount; i++ {
		pid := fmt.Sprintf("wisp_%d", i)
		c.Wisps = append(c.Wisps, pid)
		parts = append(parts, Part{
			ID: pid, Type: "sphere", Size: Vector3{0.3, 0.3, 0.3},
			Pos: c.Center, Color: Vector3{rand.Float32(), 1.0, rand.Float32()}, Locked: true,
		})
	}

	return ConstructRequestWithJoints{
		Type:        "create_construct",
		ConstructID: id,
		Parts:       parts,
	}, nil
}

func (c *WispController) Tick(dt float64, conn net.Conn) {
	c.Phase += dt * 3.0

	updates := []map[string]interface{}{}
	for i, pid := range c.Wisps {
		angle := c.Phase + float64(i)*(2*math.Pi/float64(len(c.Wisps)))
		rad := float32(2.0 + math.Sin(c.Phase*0.5)*1.0)

		offsetX := float32(math.Cos(angle)) * rad
		offsetY := float32(math.Sin(angle*0.7)) * rad
		offsetZ := float32(math.Sin(angle)) * rad

		newPos := Vector3{c.Center[0] + offsetX, c.Center[1] + offsetY, c.Center[2] + offsetZ}
		updates = append(updates, map[string]interface{}{
			"part_id":  pid,
			"position": []float32{newPos[0], newPos[1], newPos[2]},
		})
	}

	msg, _ := json.Marshal(map[string]interface{}{
		"type": "update_construct", "construct_id": c.ConstructID, "updates": updates,
	})
	writePacket(conn, msg)
}

// 6. Magic Pillar (Tall emission towers)
type PillarController struct {
	ConstructID string
}

func (c *PillarController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id

	height := 10.0 + rand.Float32()*20.0
	baseScale := 1.0 + rand.Float32()*2.0
	basePos := VecAdd(center, VecMul(normal, radius+height*0.5))

	parts := []Part{
		{
			ID: "shaft", Type: "capsule", Size: Vector3{0.4 * baseScale, height, 0},
			Pos: basePos, Color: Vector3{1, 0, 0.5}, Locked: true,
		},
	}

	return ConstructRequestWithJoints{
		Type:        "create_construct",
		ConstructID: id,
		Parts:       parts,
	}, nil
}

func (c *PillarController) Tick(dt float64, conn net.Conn) {}

// --- Main Transfiguration ---

func RunTest9() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("🌌 THE GREAT PLANETARY TRANSFIGURATION (SINGLE TARGET) 🌌")

	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		return
	}
	defer conn.Close()

	// Query individual state to find current planet
	req, _ := json.Marshal(map[string]string{"type": "query_state"})
	writePacket(conn, req)

	buf := make([]byte, 16384)
	n, _ := conn.Read(buf)
	var state StateResponse
	json.Unmarshal(buf[:n], &state)

	if state.PlanetRadius == 0 {
		fmt.Println("⚠️ No current planet found. Move closer to a planet!")
		return
	}

	p := Planet{
		Position: state.PlanetCenter,
		Radius:   state.PlanetRadius,
		Name:     "CurrentPlanet",
	}

	fmt.Printf("🎯 Targeting Planet @ %v (Radius: %.1f)\n", p.Position, p.Radius)

	var wg sync.WaitGroup
	wg.Add(1)
	go FullPlanetMakeover(&wg, p)
	wg.Wait()
}

func FullPlanetMakeover(wg *sync.WaitGroup, p Planet) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		return
	}
	defer conn.Close()

	center := Vector3{p.Position[0], p.Position[1], p.Position[2]}
	radius := p.Radius

	// DENSITY: 150 unique objects per planet
	count := 150
	controllers := []ConstructController{}

	planetScopeName := sanitizeID(p.Name)

	for i := 0; i < count; i++ {
		theta := rand.Float64() * 2 * math.Pi
		phi := math.Acos(2*rand.Float64() - 1)
		normal := Vector3{
			float32(math.Sin(phi) * math.Cos(theta)),
			float32(math.Sin(phi) * math.Sin(theta)),
			float32(math.Cos(phi)),
		}

		id := fmt.Sprintf("tg_%s_%d", planetScopeName, i)
		var c ConstructController

		r := rand.Float64()
		if r < 0.3 {
			c = &MagicTreeController{} // 30% Trees
		} else if r < 0.5 {
			c = &MagicBuildingController{} // 20% Buildings
		} else if r < 0.7 {
			c = &ManaWormController{} // 20% Animals
		} else if r < 0.85 {
			c = &WispController{} // 15% Wisps
		} else {
			c = &CrystalController{} // 15% Crystals
		}

		creq, _ := c.Spawn(id, center, normal, radius)
		data, _ := json.Marshal(creq)
		writePacket(conn, data)
		controllers = append(controllers, c)

		if i%5 == 0 {
			time.Sleep(20 * time.Millisecond)
		}
	}

	fmt.Printf("✨ %s is now a magical realm.\n", p.Name)

	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()
	last := time.Now()
	for range ticker.C {
		dt := time.Since(last).Seconds()
		last = time.Now()
		for _, c := range controllers {
			c.Tick(dt, conn)
		}
	}
}
