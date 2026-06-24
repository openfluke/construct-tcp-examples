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

// --- Concrete Controllers for Test 7 ---

// 1. Radar (Kinematic Animation)
type RadarController struct {
	BasePos     Vector3
	Normal      Vector3
	ConstructID string
	Phase       float64
	Speed       float64
}

func (c *RadarController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id
	c.Normal = normal
	c.Phase = rand.Float64() * 6.28
	c.Speed = 2.0 + rand.Float64()

	spawnHeight := radius + 0.5
	c.BasePos = VecAdd(center, VecMul(normal, spawnHeight))

	parts := []Part{
		{ID: "base", Type: "box", Size: Vector3{2, 1, 2}, Pos: c.BasePos, Color: Vector3{0.3, 0.3, 0.3}, Locked: true},
		{ID: "dish", Type: "box", Size: Vector3{4, 0.2, 0.5}, Pos: VecAdd(c.BasePos, VecMul(normal, 1.5)), Color: Vector3{1.0, 0.2, 0.2}, Locked: true},
	}

	return ConstructRequestWithJoints{
		Type:        "create_construct",
		ConstructID: id,
		Parts:       parts,
	}, nil
}

func (c *RadarController) Tick(dt float64, conn net.Conn) {
	c.Phase += c.Speed * dt

	// Kinematic Orbit Logic
	t1 := Vector3{c.Normal[1], -c.Normal[0], 0}
	if c.Normal[0] == 0 && c.Normal[1] == 0 {
		t1 = Vector3{1, 0, 0}
	}
	t1 = VecNorm(t1)
	t2 := Cross(c.Normal, t1)

	orbitRadius := float32(1.5)
	x := float32(math.Cos(c.Phase)) * orbitRadius
	z := float32(math.Sin(c.Phase)) * orbitRadius

	offset := VecAdd(VecMul(t1, x), VecMul(t2, z))
	newPos := VecAdd(c.BasePos, VecAdd(VecMul(c.Normal, 1.5), offset))

	updateReq := map[string]interface{}{
		"type":         "update_construct",
		"construct_id": c.ConstructID,
		"updates": []map[string]interface{}{
			{
				"part_id":  "dish",
				"position": []float32{newPos[0], newPos[1], newPos[2]},
			},
		},
	}
	msg, _ := json.Marshal(updateReq)
	writePacket(conn, msg)
}

// 2. Windmill (Physics Hinge + Torque)
type WindmillController struct {
	ConstructID string
	BladeID     string
}

func (c *WindmillController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id
	c.BladeID = "blades"

	spawnHeight := radius + 2.0
	basePos := VecAdd(center, VecMul(normal, spawnHeight))
	bladePos := VecAdd(basePos, VecMul(normal, 2.0)) // Top of tower

	parts := []Part{
		{ID: "tower", Type: "box", Size: Vector3{1, 4, 1}, Pos: basePos, Color: Vector3{0.6, 0.4, 0.2}, Locked: true},
		{ID: c.BladeID, Type: "box", Size: Vector3{8, 0.5, 0.5}, Pos: bladePos, Color: Vector3{0.9, 0.9, 0.9}, Locked: false, Groups: []string{"lasso_target"}},
	}

	joints := []JointDef{
		{Type: "hinge", A: "tower", B: c.BladeID, Pos: []float32{bladePos[0], bladePos[1], bladePos[2]}},
	}

	return ConstructRequestWithJoints{
		Type:        "create_construct",
		ConstructID: id,
		Parts:       parts,
		Joints:      joints,
	}, nil
}

func (c *WindmillController) Tick(dt float64, conn net.Conn) {
	updateReq := map[string]interface{}{
		"type":         "update_construct",
		"construct_id": c.ConstructID,
		"updates": []map[string]interface{}{
			{
				"part_id": c.BladeID,
				"torque":  []float32{10, 0, 0},
			},
		},
	}
	msg, _ := json.Marshal(updateReq)
	writePacket(conn, msg)
}

// 3. Lantern (Physics Chain - Dangling)
type LanternController struct {
	ConstructID string
}

func (c *LanternController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id

	spawnHeight := radius + 4.0
	basePos := VecAdd(center, VecMul(normal, radius+2.0))
	tipPos := VecAdd(center, VecMul(normal, spawnHeight))

	t1 := Vector3{normal[1], -normal[0], 0}
	if normal[0] == 0 && normal[1] == 0 {
		t1 = Vector3{1, 0, 0}
	}
	t1 = VecNorm(t1)

	armTip := VecAdd(tipPos, VecMul(t1, 2.0))
	lanternPos := VecSub(armTip, VecMul(normal, 1.0))

	parts := []Part{
		{ID: "pole", Type: "box", Size: Vector3{0.5, 4, 0.5}, Pos: basePos, Color: Vector3{0.2, 0.2, 0.2}, Locked: true},
		{ID: "arm", Type: "box", Size: Vector3{2.5, 0.5, 0.5}, Pos: VecAdd(tipPos, VecMul(t1, 1.0)), Color: Vector3{0.2, 0.2, 0.2}, Locked: true},
		{ID: "lantern", Type: "box", Size: Vector3{1, 1.5, 1}, Pos: lanternPos, Color: Vector3{1.0, 0.8, 0.2}, Locked: false, Groups: []string{"lasso_target"}},
	}

	joints := []JointDef{
		{Type: "pin", A: "arm", B: "lantern", Pos: []float32{armTip[0], armTip[1], armTip[2]}},
	}

	return ConstructRequestWithJoints{
		Type:        "create_construct",
		ConstructID: id,
		Parts:       parts,
		Joints:      joints,
	}, nil
}

func (c *LanternController) Tick(dt float64, conn net.Conn) {
	if rand.Float64() < 0.05 {
		updateReq := map[string]interface{}{
			"type":         "update_construct",
			"construct_id": c.ConstructID,
			"updates": []map[string]interface{}{
				{
					"part_id": "lantern",
					"torque":  []float32{float32(rand.NormFloat64() * 5), 0, float32(rand.NormFloat64() * 5)},
				},
			},
		}
		msg, _ := json.Marshal(updateReq)
		writePacket(conn, msg)
	}
}

// 4. Flopper (Multi-joint chaos)
type FlopperController struct {
	ConstructID string
	PartIDs     []string
}

func (c *FlopperController) Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error) {
	c.ConstructID = id
	c.PartIDs = []string{"seg1", "seg2", "seg3"}

	basePos := VecAdd(center, VecMul(normal, radius+0.5))

	parts := []Part{
		{ID: "base", Type: "box", Size: Vector3{2, 1, 2}, Pos: basePos, Color: Vector3{0.1, 0.5, 0.1}, Locked: true},
	}

	currentPos := basePos
	prevID := "base"
	joints := []JointDef{}

	for i, pid := range c.PartIDs {
		nextPos := VecAdd(currentPos, VecMul(normal, 1.2))
		parts = append(parts, Part{
			ID: pid, Type: "box", Size: Vector3{0.8, 1, 0.8}, Pos: nextPos, Color: Vector3{0.2, float32(0.5 + float64(i)*0.2), 0.2}, Locked: false, Groups: []string{"lasso_target"},
		})
		jointPos := VecAdd(currentPos, VecMul(normal, 0.6))
		joints = append(joints, JointDef{
			Type: "pin", A: prevID, B: pid, Pos: []float32{jointPos[0], jointPos[1], jointPos[2]},
		})
		currentPos = nextPos
		prevID = pid
	}

	return ConstructRequestWithJoints{
		Type:        "create_construct",
		ConstructID: id,
		Parts:       parts,
		Joints:      joints,
	}, nil
}

func (c *FlopperController) Tick(dt float64, conn net.Conn) {
	updates := []map[string]interface{}{}
	for _, pid := range c.PartIDs {
		updates = append(updates, map[string]interface{}{
			"part_id": pid,
			"torque": []float32{
				float32(rand.NormFloat64() * 10),
				float32(rand.NormFloat64() * 10),
				float32(rand.NormFloat64() * 10),
			},
		})
	}

	updateReq := map[string]interface{}{
		"type":         "update_construct",
		"construct_id": c.ConstructID,
		"updates":      updates,
	}
	msg, _ := json.Marshal(updateReq)
	writePacket(conn, msg)
}

// --- Main Runner ---

func RunTest7() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("🚀 Starting Test 7: Dynamic Planetary Defense Grid (Multi-Type Edition)")

	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Println("❌ Failed to connect to server:", err)
		return
	}
	defer conn.Close()

	// 1. Discover Planets
	fmt.Println("📡 Scanning for nearby planets...")
	req, _ := json.Marshal(map[string]string{"type": "query_nearby_planets"})
	writePacket(conn, req)

	buf := make([]byte, 65536)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("❌ Failed to read response:", err)
		return
	}

	var planetResp NearbyPlanetsResponse
	if err := json.Unmarshal(buf[:n], &planetResp); err != nil {
		fmt.Println("❌ Failed to parse planet data:", err)
		return
	}

	if len(planetResp.Planets) == 0 {
		planetResp.Planets = []Planet{{Position: []float32{0, 0, 0}, Radius: 100, Name: "Unknown"}}
	}

	fmt.Printf("✅ Found %d planets. Deploying various contraptions...\n", len(planetResp.Planets))

	var wg sync.WaitGroup
	for _, p := range planetResp.Planets {
		wg.Add(1)
		go DeployOnPlanetMixed(&wg, p)
	}

	wg.Wait()
}

func DeployOnPlanetMixed(wg *sync.WaitGroup, p Planet) {
	defer wg.Done()

	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		return
	}
	defer conn.Close()

	center := Vector3{p.Position[0], p.Position[1], p.Position[2]}
	radius := p.Radius

	constructCount := 20
	controllers := []ConstructController{}

	for i := 0; i < constructCount; i++ {
		theta := rand.Float64() * 2 * math.Pi
		phi := math.Acos(2*rand.Float64() - 1)
		normal := Vector3{
			float32(math.Sin(phi) * math.Cos(theta)),
			float32(math.Sin(phi) * math.Sin(theta)),
			float32(math.Cos(phi)),
		}

		id := fmt.Sprintf("%s_c%d", p.Name, i)
		var c ConstructController

		r := rand.Float64()
		if r < 0.25 {
			c = &RadarController{}
		} else if r < 0.5 {
			c = &WindmillController{}
		} else if r < 0.75 {
			c = &LanternController{}
		} else {
			c = &FlopperController{}
		}

		req, _ := c.Spawn(id, center, normal, radius)
		data, _ := json.Marshal(req)
		writePacket(conn, data)

		controllers = append(controllers, c)
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Printf("   >> Deployed %d mixed constructs on %s\n", constructCount, p.Name)

	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()
	lastTime := time.Now()

	for range ticker.C {
		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		lastTime = now
		for _, c := range controllers {
			c.Tick(dt, conn)
		}
	}
}
