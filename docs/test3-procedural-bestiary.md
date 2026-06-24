# Test 3: Procedural Bestiary

## Overview

**Test 3** spawns **20 procedurally generated creatures** in parallel. Each creature gets random DNA (type, colors, scale, mutations) and a behavior loop matched to its morphology.

## Run

```bash
go run . 3
```

Runs for ~120 seconds per creature, then exits.

## Creature types

| Type | Morphology | Animation |
|------|------------|-----------|
| **Worm** (0) | 4–15 linked spheres/boxes | Vertical sine torques per segment |
| **Star** (1) | Central sphere + 3–7 capsule arms | Constant yaw torque on arms |
| **Walker** (2) | Box body + 4 or 6 capsule legs | Alternating leg torques |
| **Cloud** (3) | 5–14 locked spheres in a blob | Kinematic vertical bob per sphere |
| **Totem** (4) | 3–7 stacked locked boxes | Static (no updates) |

Each creature gets a generated name like `"Swift Crimson Worm"`.

## Protocol messages

| Message | Usage |
|---------|--------|
| `create_construct` | Randomized parts + optional pin joints |
| `update_construct` | Type-specific torque or position updates (40 ms tick) |

## Highlights

- Procedural naming from dominant color channel
- Staggered spawns (100 ms apart) to avoid flooding the server
- Spawns scattered in a ~40 m × 40 m area around `(0, 4, 0)`

## Source

`test3.go` — `RunTest3()`, `StartUniqueCreature()`, `GenerateName()`
