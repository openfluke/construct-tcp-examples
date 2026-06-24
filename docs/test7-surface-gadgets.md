# Test 7: Surface Gadgets

## Overview

**Test 7** deploys a **mixed grid of surface contraptions** on every nearby planet. It discovers planets via **`query_nearby_planets`**, then spawns 20 random gadgets per planet with type-specific controllers.

## Run

```bash
go run . 7
```

Runs indefinitely (30 ms controller tick per planet goroutine).

## Gadget types

| Controller | ~25% each | Mechanism |
|------------|-----------|-----------|
| **Radar** | Kinematic | Dish orbits base on tangent plane |
| **Windmill** | Physics | Hinge joint + constant blade torque |
| **Lantern** | Physics | Pin-jointed dangling lantern, occasional random torque |
| **Flopper** | Physics | 3-segment pin chain with random multi-axis torques |

## Protocol messages

| Message | Usage |
|---------|--------|
| `query_nearby_planets` | Planet list (falls back to synthetic planet if empty) |
| `create_construct` | Spawn gadget with parts + optional `pin`/`hinge` joints |
| `update_construct` | Per-controller tick (torque or kinematic position) |

## Joint types demonstrated

- **`pin`** — lantern swing, flopper segments
- **`hinge`** — windmill blade rotation

## Placement

Random surface points via uniform sphere sampling (`theta`, `phi`) around each planet center.

## Source

`test7.go` — `RunTest7()`, `DeployOnPlanetMixed()`, controller types in same file
