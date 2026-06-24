package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Test 13: spawn one of each supported construct shape variant.
func RunTest13() {
	fmt.Println("🤖 Test 13: Rons Gone Wrong")

	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ Failed to connect to server: %v\n", err)
		return
	}
	defer conn.Close()

	// Anchor clusters just above the current planet surface (not inside it).
	q, _ := json.Marshal(map[string]string{"type": "query_state"})
	writePacket(conn, q)
	var state StateResponse
	if err := readJSONPacket(conn, &state, 2*time.Second); err != nil {
		fmt.Printf("❌ query_state failed: %v\n", err)
		return
	}
	if len(state.PlanetCenter) < 3 || len(state.PlayerPos) < 3 || state.PlanetRadius <= 0 {
		fmt.Println("⚠️ Invalid state payload from server")
		return
	}
	center := Vector3{state.PlanetCenter[0], state.PlanetCenter[1], state.PlanetCenter[2]}
	player := Vector3{state.PlayerPos[0], state.PlayerPos[1], state.PlayerPos[2]}
	up := VecNorm(VecSub(player, center))
	right, _, forward := MakeBasis(up)
	anchor := VecAdd(center, VecMul(up, state.PlanetRadius+3.2))

	const total = 300
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < total; i++ {
		// "Rons gone wrong" vibe: clustered bot swarms near current planet surface.
		cluster := i % 12
		clusterX := (float32(cluster%4) - 1.5) * 10.0
		clusterZ := (float32(cluster/4) - 1.0) * 10.0
		local := Vector3{
			clusterX + (rand.Float32()-0.5)*3.4,
			0.8 + rand.Float32()*2.0, // keep above surface
			clusterZ + (rand.Float32()-0.5)*3.4,
		}
		pos := TransformPoint(anchor, right, up, forward, local)

		shapeRoll := rand.Float64()
		typ := "box"
		size := Vector3{1.2, 1.2, 1.2}
		switch {
		case shapeRoll < 0.15:
			typ = "box"
			size = Vector3{0.6 + rand.Float32()*0.8, 0.9 + rand.Float32()*1.4, 0.6 + rand.Float32()*0.8}
		case shapeRoll < 0.30:
			typ = "sphere"
			r := 0.35 + rand.Float32()*0.6
			size = Vector3{r, r, r}
		case shapeRoll < 0.62:
			typ = "cylinder"
			size = Vector3{0.25 + rand.Float32()*0.35, 1.0 + rand.Float32()*2.0, 0}
		default:
			typ = "capsule"
			size = Vector3{0.2 + rand.Float32()*0.28, 0.9 + rand.Float32()*1.8, 0}
		}
		locked := rand.Float64() < 0.82
		// Palette: white shells + cyan/blue accents + occasional red "error" units.
		var col Vector3
		if rand.Float64() < 0.12 {
			col = Vector3{0.95, 0.18 + rand.Float32()*0.2, 0.22}
		} else if rand.Float64() < 0.50 {
			col = Vector3{0.75 + rand.Float32()*0.25, 0.90 + rand.Float32()*0.1, 1.0}
		} else {
			col = Vector3{0.35 + rand.Float32()*0.3, 0.75 + rand.Float32()*0.25, 0.95}
		}
		req := ConstructRequest{
			Type:        "create_construct",
			ConstructID: "rons_gone_wrong",
			Parts: []Part{
				{
					ID:     fmt.Sprintf("ron_%03d", i),
					Type:   typ,
					Size:   size,
					Pos:    pos,
					Color:  col,
					Locked: locked,
				},
			},
		}
		b, _ := json.Marshal(req)
		writePacket(conn, b)
		if i%6 == 0 {
			time.Sleep(25 * time.Millisecond)
		}
	}

	fmt.Printf(
		"✅ Spawned %d broken Rons above planet @ [%.1f %.1f %.1f], r=%.1f\n",
		total, center[0], center[1], center[2], state.PlanetRadius,
	)
	fmt.Println("🟢 Holding TCP session open. Press Ctrl+C to disconnect.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	fmt.Println("👋 Closing test13 connection...")
}

func readJSONPacket(conn net.Conn, v any, timeout time.Duration) error {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	defer conn.SetReadDeadline(time.Time{})

	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	length := int(binary.LittleEndian.Uint32(header))
	if length <= 0 || length > 10*1024*1024 {
		buf := make([]byte, 64*1024)
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		raw := append([]byte{}, header...)
		raw = append(raw, buf[:n]...)
		return json.Unmarshal(raw, v)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return err
	}
	return json.Unmarshal(payload, v)
}

