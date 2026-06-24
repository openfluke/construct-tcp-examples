# Test 4: Planetary Architecture

## Overview

**Test 4** builds **30 procedural structures** aligned to a synthetic planet surface. Each building uses a local tangent basis so box parts sit upright relative to the surface normal.

## Run

```bash
go run . 4
```

## What it does

1. Places buildings in a ring around origin `(0, 4, 0)`
2. Computes surface **normal** from each spawn point toward planet center `(0, 0, 0)`
3. Transforms local box coordinates into world space via `MakeBasis` + `TransformPoint`
4. Assigns building rotation from the basis (YXZ Euler)

## Building types

| Type | Structure |
|------|-----------|
| Skyscraper (0) | Wide base + stacked floors |
| Helix (1) | Rotating offset floors around a central axis |
| Pyramid (2) | Shrinking square layers |
| Random (default) | Scattered boxes at random local offsets |

Heights vary from 5–19 floors. Each building gets a random name like `"Neo Skyscraper (H:12)"`.

## Protocol messages

| Message | Usage |
|---------|--------|
| `create_construct` | Multi-part box buildings (no joints) |

Parts use **`Rot`** aligned to the surface — important for curved-planet placement.

## Notes

Uses a **synthetic planet center** at origin for normal calculation. For real multiverse planets, use **`query_state`** (see test 5, 8, 13) to get live center/radius.

## Source

`test4.go` — `RunTest4()`, `StartBuildingOnPlanet()`, `GenerateBuildingName()`
