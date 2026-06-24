package main

import (
	"fmt"
	"os"
	"strconv"
)

// construct-tcp-examples — Construct TCP integration harness (default: example 13).
//
// Run: go run . [n]   host must listen on 127.0.0.1:17000
func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printUsage()
		return
	}

	testNum := 13
	if len(os.Args) > 1 {
		if num, err := strconv.Atoi(os.Args[1]); err == nil {
			testNum = num
		}
	}

	switch testNum {
	case 1:
		RunTest1()
	case 2:
		RunTest2()
	case 3:
		RunTest3()
	case 4:
		RunTest4()
	case 5:
		RunTest5()
	case 6:
		RunTest6()
	case 7:
		RunTest7()
	case 8:
		RunTest8()
	case 9:
		RunTest9()
	case 10:
		RunTest10()
	case 11:
		RunTest11()
	case 12:
		RunTest12()
	case 13:
		RunTest13()
	case 14:
		RunLoadTest()
	default:
		fmt.Println("Unknown example number. Run: go run . -h")
	}
}

func printUsage() {
	fmt.Println(`construct-tcp-examples — Construct TCP example harness

Usage:
  go run .         Run example 13 (default)
  go run . <n>     Run example 1–14
  go run . -h      This help

Requires a Construct TCP host on 127.0.0.1:17000.

Examples:
  1   Animated pin-jointed skeleton (torque loop)
  2   Multi-client snake bots
  3   Procedural creature swarm
  4   Procedural buildings on planet surface
  5   Auto-discovery satellite
  6   Query loop — state, planets, players, performance
  7   Surface gadgets (radar, windmill, …)
  8   Multi-bubble discovery spawn
  9   Magic trees + extended controllers
  10  Swarm RL cubes (needs loom — see README)
  11  Walking skeleton RL (needs loom — see README)
  12  Swarm skeletons at bubble waypoints
  13  Shape showcase — default (go run .)
  14  Load test + live WebSocket dashboard

Do not run a single file (go run test1.go) — types live in shared.go.`)
}
