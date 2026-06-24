# Test 13: Shape Showcase (default)

## Overview

**Test 13** — *Rons Gone Wrong* — is the **default example** (`go run .`). It queries live planet state, then spawns **300 random constructs** clustered above the current planet surface, exercising every supported part shape.

## Run

```bash
go run .       # default
go run . 13
```

Holds the TCP session open until **Ctrl+C**.

## What it does

1. **`query_state`** → planet center, radius, player position
2. Build surface anchor: `planet_center + up × (radius + 3.2)` where `up` is player − center
3. Spawn 300 parts in 12 clusters near the anchor (local XZ grid + jitter)
4. Random shape, size, color, and locked flag per part

## Shape distribution

| Shape | ~Probability | Size notes |
|-------|--------------|------------|
| `box` | 15% | Random width/height/depth |
| `sphere` | 15% | Uniform radius |
| `cylinder` | 32% | Radius + height |
| `capsule` | 38% | Radius + height |

~82% of parts are **locked** (kinematic decorations); remainder simulate freely.

## Color palette

White/cyan/blue bot shells with occasional red "error" units — the "Rons gone wrong" look.

## Protocol messages

| Message | Usage |
|---------|--------|
| `query_state` | Anchor spawn to real planet (length-prefixed read via `readJSONPacket`) |
| `create_construct` | One part per request, shared construct id `rons_gone_wrong` |

## Why this is the default

- Works with any host that returns valid planet state
- Demonstrates all four part types in one run
- Shows surface-relative spawning (not hard-coded origin)

## Source

`test13.go` — `RunTest13()`, `readJSONPacket()`
