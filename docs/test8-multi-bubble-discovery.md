# Test 8: Multi-Bubble Discovery

## Overview

**Test 8** uses **`query_state`** to discover **bubble waypoints** on the current planet, then spawns a visual marker group at each bubble — one sphere on the bubble, one 5 m above, and a connector capsule between them.

## Run

```bash
go run . 8
```

Exits after spawning (one-shot). Requires bubbles in the server state.

## What it does

1. **`query_state`** → read `planet_center`, `planet_radius`, `bubbles[]`
2. For each bubble:
   - Compute **up** vector (bubble position − planet center, normalized)
   - Spawn construct `bubble_group_{index}` with three locked parts:
     - `on_bubble` — green sphere at bubble position
     - `above_bubble` — orange sphere 5 m along up
     - `connector` — gray capsule bridging the two

## Protocol messages

| Message | Usage |
|---------|--------|
| `query_state` | Bubble list + planet context |
| `create_construct` | Three-part marker per bubble |

## Requirements

- Host must populate `bubbles` in `state_response`
- If zero bubbles: prints warning and exits

## Source

`test8.go` — `RunTest8()`
