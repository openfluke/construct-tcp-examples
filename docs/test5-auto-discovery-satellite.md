# Test 5: Auto-Discovery Satellite

## Overview

**Test 5** demonstrates **`query_state`**-driven spawning. It reads live planet center, radius, and player position from the server, then launches five orbiting satellites whose paths are computed from that state.

## Run

```bash
go run . 5
```

Best with a multiverse-capable host that returns valid `state_response` data.

## What it does

1. Connects and sends **`query_state`**
2. Parses `state_response`: `player_pos`, `planet_center`, `planet_radius`
3. Spawns 5 satellites (each on its own connection) at increasing orbit distances
4. Animates orbit for 120 seconds using kinematic **position** updates on core + solar panels

## Satellite structure

Each satellite is three locked box parts:

- `core` — 2×2×2 m body
- `panel_l` / `panel_r` — 4×0.2×3 m panels offset ±3 m on X

Orbit radius: `planet_radius + 50 + id×10` meters.

## Protocol messages

| Message | Usage |
|---------|--------|
| `query_state` | Discover planet + player |
| `create_construct` | Spawn satellite |
| `update_construct` | Kinematic orbit (30 ms tick) |

## Source

`test5.go` — `RunTest5()`, `SpawnSatellite()`
