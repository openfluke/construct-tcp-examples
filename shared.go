// Package main — Construct TCP client examples (PrimeCraft wire format).
//
//   • Client → server: 4-byte little-endian length + UTF-8 JSON
//   • Server → client: raw JSON (query_performance is length-prefixed)
//
// Default host: 127.0.0.1:17000
package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"net"
)

// --- Common Network Structs ---

type Vector3 [3]float32

// ConstructServerAddress is the default Construct TCP host.
const ConstructServerAddress = "127.0.0.1:17000"

// PlanetGridSpacingMeters / PlanetVerticalSpacingMeters — multiverse grid spacing (example 6+).
const PlanetGridSpacingMeters float32 = 800
const PlanetVerticalSpacingMeters float32 = 800

// PlanetCellWorldCenter converts multiverse cell indices to approximate world meters (no jitter).
func PlanetCellWorldCenter(ix, iy, iz int) Vector3 {
	return Vector3{
		float32(ix) * PlanetGridSpacingMeters,
		float32(iy) * PlanetVerticalSpacingMeters,
		float32(iz) * PlanetGridSpacingMeters,
	}
}

type Basis struct {
	X, Y, Z Vector3
}

func GetBasis(normal Vector3) Basis {
	x, y, z := MakeBasis(normal)
	return Basis{x, y, z}
}

func GetEuler(b Basis) Vector3 {
	// YXZ ordering (Godot standard)
	m00, _, m02 := float64(b.X[0]), float64(b.Y[0]), float64(b.Z[0])
	m10, m11, m12 := float64(b.X[1]), float64(b.Y[1]), float64(b.Z[1])
	m20, _, m22 := float64(b.X[2]), float64(b.Y[2]), float64(b.Z[2])

	var ex, ey, ez float64
	m12 = math.Max(-1.0, math.Min(1.0, m12))
	ex = math.Asin(-m12)

	if math.Abs(m12) < 0.99999 {
		ey = math.Atan2(m02, m22)
		ez = math.Atan2(m10, m11)
	} else {
		ey = math.Atan2(-m20, m00)
		ez = 0
	}
	return Vector3{RadToDeg(ex), RadToDeg(ey), RadToDeg(ez)}
}

type Part struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Size         Vector3  `json:"size"`
	Pos          Vector3  `json:"pos"`
	Rot          Vector3  `json:"rot,omitempty"`
	Locked       bool     `json:"locked"`
	LockRotation bool     `json:"lock_rotation"`
	Color        Vector3  `json:"color"`
	Groups       []string `json:"groups,omitempty"`
	IsHorizontal bool     `json:"is_horizontal,omitempty"`
}

type Joint struct {
	Type string  `json:"type"`
	A    string  `json:"a"`
	B    string  `json:"b"`
	Pos  Vector3 `json:"pos"`
}

type JointDef struct {
	Type string    `json:"type"`
	A    string    `json:"a"`
	B    string    `json:"b"`
	Pos  []float32 `json:"pos"`
}

type ConstructRequest struct {
	Type        string  `json:"type"`
	ConstructID string  `json:"construct_id"`
	Parts       []Part  `json:"parts"`
	Joints      []Joint `json:"joints,omitempty"`
}

type ConstructRequestWithJoints struct {
	Type        string     `json:"type"`
	ConstructID string     `json:"construct_id"`
	Parts       []Part     `json:"parts"`
	Joints      []JointDef `json:"joints,omitempty"`
}

type PartUpdate struct {
	PartID          string   `json:"part_id"`
	Torque          *Vector3 `json:"torque,omitempty"`
	AngularVelocity *Vector3 `json:"angular_velocity,omitempty"`
	LinearVelocity  *Vector3 `json:"linear_velocity,omitempty"`
	Position        *Vector3 `json:"position,omitempty"`
	Rotation        *Vector3 `json:"rotation,omitempty"`
}

type UpdateRequest struct {
	Type        string       `json:"type"`
	ConstructID string       `json:"construct_id"`
	Updates     []PartUpdate `json:"updates"`
}

type Planet struct {
	Position []float32   `json:"Position"`
	Radius   float32     `json:"Radius"`
	Biome    interface{} `json:"Biome,omitempty"`
	Seed     string      `json:"Seed,omitempty"`
	Name     string      `json:"Name"`
}

type BubbleInfo struct {
	Index   int       `json:"index"`
	Pos     []float32 `json:"pos"`
	Visible bool      `json:"visible"`
}

type StateResponse struct {
	Type         string       `json:"type"`
	PlayerPos    []float32    `json:"player_pos"`
	PlanetCenter []float32    `json:"planet_center"`
	PlanetRadius float32      `json:"planet_radius"`
	Bubbles      []BubbleInfo `json:"bubbles"`
}

type Player struct {
	Position []float32 `json:"Position"`
	Rotation []float32 `json:"Rotation"`
	UUID     string    `json:"UUID"`
	Index    int       `json:"Index"`
	IsUser   bool      `json:"IsUser"`
	Name     string    `json:"Name"`
}

type NearbyPlanetsResponse struct {
	Type    string   `json:"type"`
	Planets []Planet `json:"planets"`
	Count   int      `json:"count"`
}

type PlayersResponse struct {
	Type    string   `json:"type"`
	Players []Player `json:"players"`
	Count   int      `json:"count"`
}

type PerformanceResponse struct {
	Type                         string  `json:"type"`
	EngineFPS                    float64 `json:"engine_fps"`
	TimeFPS                      float64 `json:"time_fps"`
	TimeProcess                  float64 `json:"time_process"`
	TimePhysicsProcess           float32 `json:"time_physics_process"` // Keep as matches server double
	TimeNavigationProcess        float64 `json:"time_navigation_process"`
	MemoryStatic                 float64 `json:"memory_static"`
	MemoryStaticMax              float64 `json:"memory_static_max"`
	MemoryMsgBufMax              float64 `json:"memory_msg_buf_max"`
	ObjectCount                  float64 `json:"object_count"`
	ObjectResourceCount          float64 `json:"object_resource_count"`
	ObjectNodeCount              float64 `json:"object_node_count"`
	ObjectOrphanNodeCount        float64 `json:"object_orphan_node_count"`
	RenderTotalObjectsInFrame    float64 `json:"render_total_objects_in_frame"`
	RenderTotalPrimitivesInFrame float64 `json:"render_total_primitives_in_frame"`
	RenderTotalDrawCallsInFrame  float64 `json:"render_total_draw_calls_in_frame"`
	RenderVideoMemUsed           float64 `json:"render_video_mem_used"`
	RenderTextureMemUsed         float64 `json:"render_texture_mem_used"`
	RenderBufferMemUsed          float64 `json:"render_buffer_mem_used"`
	Physics3DActiveObjects       float64 `json:"physics_3d_active_objects"`
	Physics3DCollisionPairs      float64 `json:"physics_3d_collision_pairs"`
	Physics3DIslandCount         float64 `json:"physics_3d_island_count"`
	AudioOutputLatency           float64 `json:"audio_output_latency"`
	NavActiveMaps                float64 `json:"nav_active_maps"`
	NavRegionCount               float64 `json:"nav_region_count"`
	NavAgentCount                float64 `json:"nav_agent_count"`
	NavLinkCount                 float64 `json:"nav_link_count"`
	NavPolygonCount              float64 `json:"nav_polygon_count"`
	NavEdgeCount                 float64 `json:"nav_edge_count"`
	NavEdgeMergeCount            float64 `json:"nav_edge_merge_count"`
	NavEdgeConnectionCount       float64 `json:"nav_edge_connection_count"`
	NavEdgeFreeCount             float64 `json:"nav_edge_free_count"`
}

// --- Networking Helper ---

func writePacket(conn net.Conn, data []byte) {
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, uint32(len(data)))
	conn.Write(header)
	conn.Write(data)
}

// --- Math Utilities ---

func MakeBasis(normal Vector3) (Vector3, Vector3, Vector3) {
	up := Vector3{0, 1, 0}
	if math.Abs(float64(normal[1])) > 0.99 {
		up = Vector3{1, 0, 0}
	}
	right := Cross(up, normal)
	right = Normalize(right)
	forward := Cross(normal, right)
	return right, normal, forward
}

func TransformPoint(origin Vector3, bX, bY, bZ Vector3, local Vector3) Vector3 {
	return Vector3{
		origin[0] + bX[0]*local[0] + bY[0]*local[1] + bZ[0]*local[2],
		origin[1] + bX[1]*local[0] + bY[1]*local[1] + bZ[1]*local[2],
		origin[2] + bX[2]*local[0] + bY[2]*local[1] + bZ[2]*local[2],
	}
}

func Cross(a, b Vector3) Vector3 {
	return Vector3{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func Normalize(v Vector3) Vector3 {
	mag := float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
	if mag == 0 {
		return v
	}
	return Vector3{v[0] / mag, v[1] / mag, v[2] / mag}
}

func RadToDeg(rad float64) float32 {
	return float32(rad * 180.0 / math.Pi)
}

func Add(a, b Vector3) Vector3 {
	return Vector3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func Mul(v Vector3, s float32) Vector3 {
	return Vector3{v[0] * s, v[1] * s, v[2] * s}
}

// VecAdd and VecMul for compatibility with some older tests
func VecAdd(a, b Vector3) Vector3         { return Add(a, b) }
func VecMul(v Vector3, s float32) Vector3 { return Mul(v, s) }

func VecSub(a, b Vector3) Vector3 { return Vector3{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func VecLen(v Vector3) float32    { return float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]))) }
func VecNorm(v Vector3) Vector3 {
	l := VecLen(v)
	if l == 0 {
		return Vector3{0, 1, 0}
	}
	return VecMul(v, 1.0/l)
}

func VecDot(a, b Vector3) float32 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

func ScaleVec(v Vector3, s float32) Vector3 {
	return Vector3{v[0] * s, v[1] * s, v[2] * s}
}

func RandomColor() Vector3 {
	return Vector3{rand.Float32(), rand.Float32(), rand.Float32()}
}

func sanitizeID(s string) string {
	res := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			res += string(r)
		} else {
			res += "_"
		}
	}
	return res
}

// --- Interfaces ---

type ConstructController interface {
	Spawn(id string, center Vector3, normal Vector3, radius float32) (ConstructRequestWithJoints, error)
	Tick(dt float64, conn net.Conn)
}
