# Test 1: Pin-Jointed Skeleton

## Overview

**Test 1** is the introductory Construct TCP example. It spawns a humanoid skeleton made of capsule parts connected with **pin joints**, then drives a simple animation loop with `update_construct`.

## Run

```bash
go run . 1
```

Requires a Construct TCP host on `127.0.0.1:17000`.

## What it does

1. Connects to the construct server
2. Sends **`create_construct`** with 10 capsule parts (torso, head, arms, legs) and 9 pin joints
3. Runs a 20 ms tick loop that:
   - Bobs the torso with a sine wave (kinematic **position** override)
   - Applies alternating **torque** to left/right forearms

## Protocol messages

| Message | Usage |
|---------|--------|
| `create_construct` | Spawn skeleton with parts + pin joints |
| `update_construct` | Per-tick torso position + arm torques |

## Part types & joints

- **Parts:** `capsule` only
- **Joints:** `pin` (neck, shoulders, elbows, hips, knees)
- **Groups:** `lasso_target` on all parts (grabbable in compatible hosts)

## Spawn coordinates

Spawns near world origin `(0, 4, 0)`. Many construct hosts use a local physics sandbox at this scale rather than full multiverse cell coordinates.

## Source

`test1.go` — `RunTest1()`
