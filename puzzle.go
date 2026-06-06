package main

// Color represents a grid cell's color. 0 means empty.
type Color int

const (
	Empty Color = 0
	// Wall marks a cell that is not part of the board (a hole), letting the
	// playable area take non-rectangular shapes within its bounding box.
	Wall Color = -1
)

// Point is a coordinate on the grid.
type Point struct {
	X, Y int
}

func (p Point) Add(d Point) Point {
	return Point{p.X + d.X, p.Y + d.Y}
}

// Up, down, left, right movement directions.
var Dirs = []Point{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

// Puzzle holds the original problem data and does not change during solving.
type Puzzle struct {
	W, H       int
	Endpoints  map[Color][2]Point
	Colors     []Color
	ColorNames map[Color]string // original characters when parsed (e.g., "R", "G")
	playable   []bool           // index y*W+x; false = Wall (not part of the board)
}

// NewPuzzle creates a Puzzle from a 2D grid.
// grid[y][x]: 0 = empty, -1 (Wall) = hole, positive = color index.
// names optionally provides a color index → display character mapping.
func NewPuzzle(grid [][]Color, names ...map[Color]string) *Puzzle {
	h := len(grid)
	w := len(grid[0])
	seen := map[Color]int{}
	endpoints := map[Color][2]Point{}
	var colors []Color
	playable := make([]bool, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := grid[y][x]
			if c == Wall {
				continue // playable stays false
			}
			playable[y*w+x] = true
			if c == Empty {
				continue
			}
			p := Point{x, y}
			pair := endpoints[c]
			if seen[c] == 0 {
				pair[0] = p
				colors = append(colors, c)
			} else {
				pair[1] = p
			}
			endpoints[c] = pair
			seen[c]++
		}
	}

	pz := &Puzzle{
		W:         w,
		H:         h,
		Endpoints: endpoints,
		Colors:    colors,
		playable:  playable,
	}
	if len(names) > 0 {
		pz.ColorNames = names[0]
	}
	return pz
}

// InBounds reports whether the point is within the bounding rectangle.
func (pz *Puzzle) InBounds(p Point) bool {
	return p.X >= 0 && p.X < pz.W && p.Y >= 0 && p.Y < pz.H
}

// Playable reports whether p is an in-bounds board cell (not a Wall).
func (pz *Puzzle) Playable(p Point) bool {
	return pz.playable[p.Y*pz.W+p.X]
}

// NumCells returns the number of playable (non-Wall) cells — the count a full
// solution must fill.
func (pz *Puzzle) NumCells() int {
	n := 0
	for _, ok := range pz.playable {
		if ok {
			n++
		}
	}
	return n
}

// Open reports whether p is in bounds and a playable board cell. Solvers use
// this for neighbor traversal so Wall cells are never visited or connected.
func (pz *Puzzle) Open(p Point) bool {
	return pz.InBounds(p) && pz.playable[p.Y*pz.W+p.X]
}
