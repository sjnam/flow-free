package main

import "fmt"

const reset = "\033[0m"

var letterANSI = map[byte]string{
	'R': "\033[31m",
	'B': "\033[34m",
	'G': "\033[32m",
	'Y': "\033[33m",
	'M': "\033[35m",
	'C': "\033[36m",
	'W': "\033[37m",
	'P': "\033[95m",
	'O': "\033[91m",
	'A': "\033[92m",
	'D': "\033[94m",
	'V': "\033[35m",
}

var indexANSI = []string{
	"",
	"\033[31m", // 1
	"\033[34m", // 2
	"\033[32m", // 3
	"\033[33m", // 4
	"\033[35m", // 5
	"\033[36m", // 6
	"\033[37m", // 7
	"\033[91m", // 8
	"\033[94m", // 9
}

var indexLabel = []string{".", "R", "B", "G", "Y", "M", "C", "W", "r", "b"}

func colorName(pz *Puzzle, c Color) string {
	if pz != nil && pz.ColorNames != nil {
		if name, ok := pz.ColorNames[c]; ok {
			return name
		}
	}
	idx := int(c)
	if idx >= 0 && idx < len(indexLabel) {
		return indexLabel[idx]
	}
	return "?"
}

func colorize(pz *Puzzle, c Color, s string) string {
	var code string
	name := colorName(pz, c)
	if len(name) > 0 {
		if v, ok := letterANSI[name[0]]; ok {
			code = v
		}
	}
	if code == "" {
		idx := int(c)
		if idx > 0 && idx < len(indexANSI) {
			code = indexANSI[idx]
		}
	}
	if code == "" {
		return s
	}
	return code + s + reset
}

// PrintPuzzle prints the initial puzzle state (endpoints only).
func PrintPuzzle(pz *Puzzle, initGrid [][]Color) {
	fmt.Printf("Puzzle (%d x %d), %d colors\n", pz.W, pz.H, len(pz.Colors))
	printGrid(pz, pz.W, pz.H, func(x, y int) Color {
		return initGrid[y][x]
	})
}

// PrintState prints the full solution state.
// Path cells are displayed as color characters, same as endpoints.
func PrintState(s *State) {
	printGrid(s.Puzzle, s.Puzzle.W, s.Puzzle.H, func(x, y int) Color {
		return s.Grid[y][x]
	})
	fmt.Printf("Filled: %d/%d\n", s.Filled, s.Puzzle.W*s.Puzzle.H)
}

// printGrid prints the grid.
// getCell(x, y) returns the cell's color (if Empty=0, prints '.').
// Border width is exactly 3 chars per cell (─ ─ ─).
func printGrid(pz *Puzzle, w, h int, getCell func(x, y int) Color) {
	border := func(left, mid, right string) {
		fmt.Print(left)
		for x := 0; x < w; x++ {
			fmt.Print(mid)
		}
		fmt.Println(right)
	}

	border("┌", "───", "┐")

	for y := 0; y < h; y++ {
		fmt.Print("│")
		for x := 0; x < w; x++ {
			c := getCell(x, y)
			if c == Empty {
				fmt.Print(" . ")
			} else {
				ch := colorName(pz, c)
				fmt.Print(" " + colorize(pz, c, ch) + " ")
			}
		}
		fmt.Println("│")
	}

	border("└", "───", "┘")
}
