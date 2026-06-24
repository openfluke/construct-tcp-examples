# construct-tcp-examples

**Example integration harness for the Construct TCP simulation API.**

Apache-2.0 Go clients that connect to a **Construct TCP host** on `127.0.0.1:17000`, spawn rigid-body constructs, and drive the simulation over JSON. This repo is the public home of the old **`construct_tests`** numbered examples (`test1` … `test13`, plus load test **14**).

**Default:** example **13** — spawns a showcase of every supported part shape on the planet surface.

License: **Apache-2.0** (see [LICENSE](LICENSE)).

---

## Quick start

### 1. Start a Construct TCP host

You need a server listening on **`127.0.0.1:17000`** (Construct TCP / PrimeCraft wire format). Point your host at the multiverse scenario if you want planet queries (example 6) and surface-relative spawns (example 13).

### 2. Run an example

```bash
git clone https://github.com/openfluke/construct-tcp-examples.git
cd construct-tcp-examples
go run .            # same as go run . 13
go run . 13         # shape showcase (default)
go run . -h         # all example numbers
```

Run the **whole module** (`go run . N`). Do not `go run test13.go` alone — types live in `shared.go`.

---

## Example 13 (default)

**`go run . 13`** — *Rons Gone Wrong*: shape showcase.

1. Connects to `:17000`
2. **`query_state`** — reads planet center, radius, player position
3. Spawns clusters of **box**, **sphere**, **capsule**, **cylinder** parts just above the surface (anchored to live server state, not hard-coded world origin)
4. Leaves constructs simulating until you Ctrl+C

This is the best “hello world” for **create_construct** with real planet context. See [docs/test13-shape-showcase.md](docs/test13-shape-showcase.md).

---

## All examples

| # | Command | Summary | Docs |
|---|---------|---------|------|
| 1 | `go run . 1` | Pin-jointed skeleton + torque animation | [test1](docs/test1-pin-jointed-skeleton.md) |
| 2 | `go run . 2` | Multi-client snake / tower / jellyfish | [test2](docs/test2-multi-client-constructs.md) |
| 3 | `go run . 3` | Procedural creature swarm | [test3](docs/test3-procedural-bestiary.md) |
| 4 | `go run . 4` | Procedural buildings on surface | [test4](docs/test4-planetary-architecture.md) |
| 5 | `go run . 5` | Auto-discovery satellite | [test5](docs/test5-auto-discovery-satellite.md) |
| 6 | `go run . 6` | Query loop — nearby planets + players | [test6](docs/test6-universal-query.md) |
| 7 | `go run . 7` | Surface gadgets (radar, windmill, …) | [test7](docs/test7-surface-gadgets.md) |
| 8 | `go run . 8` | Multi-bubble discovery spawn | [test8](docs/test8-multi-bubble-discovery.md) |
| 9 | `go run . 9` | Magic trees + controllers | [test9](docs/test9-planetary-transfiguration.md) |
| 10 | `go run . 10` | Swarm RL cubes (needs **loom v0.80**) | [test10](docs/test10-swarm-rl.md) |
| 11 | `go run . 11` | Walking skeleton RL (needs **loom v0.80**) | [test11](docs/test11-walking-rl.md) |
| 12 | `go run . 12` | Swarm skeletons at bubbles | [test12](docs/test12-swarm-skeletons.md) |
| **13** | **`go run . 13`** | **Shape showcase (default)** | [test13](docs/test13-shape-showcase.md) |
| 14 | `go run . 14` | Load test + WebSocket monitor (`:3000`) | — |

---

## Protocol

### Transport

- **Address:** `127.0.0.1:17000` (`ConstructServerAddress` in `shared.go`)
- **Client → server:** 4-byte little-endian length + UTF-8 JSON
- **Server → client:** raw JSON (`query_performance` may be length-prefixed)

Compatible with **PrimeCraft** `ConstructServer.cs` framing.

### Common messages

| `type` | Purpose |
|--------|---------|
| `create_construct` | Parts + optional pin/hinge joints |
| `update_construct` | Torques, velocities, kinematic overrides |
| `query_state` | Player, planet center/radius, bubbles |
| `query_nearby_planets` | Planet list (multiverse) |
| `query_players` | Player slots |
| `query_performance` | FPS, body counts |

**Part types:** `box`, `sphere`, `capsule`, `cylinder`.

### Minimal create

```go
conn, _ := net.Dial("tcp", "127.0.0.1:17000")
req := map[string]any{
    "type":         "create_construct",
    "construct_id": "hello",
    "parts": []map[string]any{{
        "id": "box1", "type": "box",
        "size": []float32{1, 1, 1},
        "pos":  []float32{0, 4, 0},
        "color": []float32{1, 0, 0},
    }},
}
body, _ := json.Marshal(req)
writePacket(conn, body) // shared.go
```

---

## Dependencies

| Package | Version | Examples |
|---------|---------|----------|
| Go | 1.26+ | all |
| [loom](https://github.com/openfluke/loom) | **v0.80** | 10–11 only (`poly_adapter.go`) |
| Fiber + WebSocket | v2 | 14 only |

```bash
go mod download
go run . 13
```

Examples **1–9**, **12–13** have no loom dependency.

---

## Layout

```
construct-tcp-examples/
  main.go           go run . [n]  — default 13
  shared.go         protocol + writePacket
  test1.go … test13.go
  loadtesting.go    example 14
  poly_adapter.go         type aliases → internal/adapter
  internal/adapter/       loom CPU-MC network wrapper
  test/                   go test ./test/...
  docs/             Per-example notes (test1–test13)
```

---

## Contributing

Keep historical `testN.go` names. New example: add file, register in `main.go`, document in this README.
