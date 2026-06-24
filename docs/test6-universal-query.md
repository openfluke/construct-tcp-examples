# Test 6: Universal Query Loop

## Overview

**Test 6** is a **read-only diagnostic client**. It connects once and polls the server every 2 seconds, printing nearby planets and connected players. Useful for verifying multiverse discovery and player slot reporting.

## Run

```bash
go run . 6
```

Press **Ctrl+C** to disconnect.

## What it does

Every 2 seconds on a single TCP connection:

1. **`query_nearby_planets`** → prints planet name, seed, position, radius
2. **`query_players`** → prints player name, index, position, USER vs AI flag

## Protocol messages

| Message | Response type |
|---------|---------------|
| `query_nearby_planets` | `nearby_planets_response` |
| `query_players` | `players_response` |

Server responses are **raw JSON** (not length-prefixed on the client read path). This test assumes one synchronous read per query.

## Requirements

- Multiverse host with planet list populated
- Player slots initialized (dev players or user)

Does **not** spawn constructs.

## Source

`test6.go` — `RunTest6()`
