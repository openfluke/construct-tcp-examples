# Test 9: Planetary Transfiguration

## Overview

**Test 9** is a large-scale **planet decoration** demo. It queries the **current planet** via **`query_state`**, then spawns **150 mixed magical objects** scattered across the surface with animated controllers.

## Run

```bash
go run . 9
```

Runs indefinitely after spawn (30 ms tick loop).

## Object types

| Controller | Weight | Description |
|------------|--------|-------------|
| **MagicTree** | 30% | Capsule trunk + 5 glowing sphere leaves |
| **MagicBuilding** | 20% | Tapering neon-trim skyscraper stack |
| **ManaWorm** | 20% | Pin-jointed segmented worm with sine torques |
| **Wisp** | 15% | Anchor + 4 orbiting locked spheres (kinematic) |
| **Crystal** | 15% | Floating locked crystal + glow sphere |

## Protocol messages

| Message | Usage |
|---------|--------|
| `query_state` | Current planet center + radius |
| `create_construct` | Surface-aligned or floating constructs |
| `update_construct` | Worm torques, wisp orbit positions |

## Placement

Uniform random points on the planet sphere (same sampling as test 7). Parts use **`GetBasis`** / **`TransformPoint`** for surface-aligned trees and buildings.

## Source

`test9.go` — `RunTest9()`, `FullPlanetMakeover()`, controller types in same file
