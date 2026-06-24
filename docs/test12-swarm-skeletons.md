# Test 12: Swarm Skeletons

## Overview

**Test 12** spawns **pin-jointed skeletons at every bubble waypoint** on the current planet. For each bubble it places three skeletons in a ring 8 m from the bubble center, offset above the surface.

## Run

```bash
go run . 12
```

Stays connected indefinitely after spawn (`select {}`).

## What it does

1. **`query_state`** → read `bubbles`, `planet_center`
2. For each bubble × 3 skeletons:
   - Compute surface **up** from bubble toward planet center
   - Build ring offset using `MakeBasis` + `TransformPoint`
   - Spawn full humanoid skeleton (same topology as test 1)
3. Parts tagged `lasso_target` + `skeleton` for grabbable hosts

## Skeleton structure

10 capsule parts, 9 pin joints — torso, head, arms (horizontal capsules), legs. Same joint graph as test 1 with different colors.

## Protocol messages

| Message | Usage |
|---------|--------|
| `query_state` | Bubble positions |
| `create_construct` | One skeleton per spawn point |

No animation loop — skeletons are physics-only after spawn.

## Requirements

- Non-empty `bubbles` in `state_response`

Total skeletons: `3 × bubble_count`.

## Source

`test12.go` — `RunTest12()`, `createSkeletonGrabbable()`
