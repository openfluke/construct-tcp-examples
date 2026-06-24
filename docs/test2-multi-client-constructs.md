# Test 2: Multi-Client Constructs

## Overview

**Test 2** stress-tests **multiple simultaneous TCP clients**, each owning its own construct. Four goroutines connect in parallel and run independent animation loops for ~60 seconds.

## Run

```bash
go run . 2
```

## What it does

Spawns four constructs near the origin, each on its **own TCP connection**:

| Client | Construct | Behavior |
|--------|-----------|----------|
| SnakeBot 1 | Pin-jointed snake (sphere head + 8 box segments) | Head bob + segment wave torques |
| BoxTower 1 | Stacked boxes on locked base | Periodic sideways torque on middle block |
| Jellyfish 1 | Locked bell + 6 pin-jointed tentacle chains | Radial pulsing tentacle torques |
| SnakeBot 2 | Second snake at a different offset | Same as SnakeBot 1 |

## Protocol messages

| Message | Usage |
|---------|--------|
| `create_construct` | One per client connection |
| `update_construct` | Per-client animation loop (30–500 ms tick rates) |

## Highlights

- Demonstrates **one construct per connection** (typical server model)
- Mix of **pin joints**, **locked** parts, and **torque**-driven motion
- Uses `sync.WaitGroup` to wait for all clients to finish

## Source

`test2.go` — `RunTest2()`, `StartSnakeBot()`, `StartBoxTower()`, `StartJellyfish()`
