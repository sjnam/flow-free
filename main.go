package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func run(name string, grid [][]Color, names map[Color]string) {
	fmt.Printf("=== %s ===\n", name)
	pz := NewPuzzle(grid, names)
	PrintPuzzle(pz, grid)

	fmt.Println("\nSolving...")
	start := time.Now()
	result, calls := SolveCSP(pz)
	elapsed := time.Since(start)

	fmt.Printf("[csp] Calls: %d  Elapsed: %v\n\n", calls, elapsed)
	if result == nil {
		fmt.Println("No solution found.")
		return
	}
	PrintState(result)
	fmt.Println()
}

const usage = `Usage:
  go run . puzzle.txt   read puzzle from file

Solver: degree-constraint CSP with AC-3 propagation.

Puzzle format (puzzle.txt example):
  # comments start with '#'
  R . . . B
  . . . . .
  . G . B .
  . . . . .
  R . G . .

  Rules:
  - letters (upper/lower): color endpoints (exactly 2 per color)
  - '.' or '0': empty cell
  - space-separated ("R . B") or packed ("R.B") are both accepted
`

func main() {
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Print(usage)
		return
	}

	var (
		grid  [][]Color
		names map[Color]string
		err   error
	)

	if args[0] == "-" {
		grid, names, err = ReadPuzzle(os.Stdin)
	} else {
		f, openErr := os.Open(args[0])
		if openErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", openErr)
			os.Exit(1)
		}
		defer f.Close()
		grid, names, err = ReadPuzzle(f)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	run(args[0], grid, names)
}
