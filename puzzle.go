package main

// Color represents a grid cell's color. 0 means empty.
type Color int

const Empty Color = 0

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
}

// NewPuzzle creates a Puzzle from a 2D grid.
// grid[y][x] of 0 means empty; positive values are color indices.
// names optionally provides a color index → display character mapping.
func NewPuzzle(grid [][]Color, names ...map[Color]string) *Puzzle {
	h := len(grid)
	w := len(grid[0])
	seen := map[Color]int{}
	endpoints := map[Color][2]Point{}
	var colors []Color

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := grid[y][x]
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
	}
	if len(names) > 0 {
		pz.ColorNames = names[0]
	}
	return pz
}

// InBounds reports whether the point is within the grid.
func (pz *Puzzle) InBounds(p Point) bool {
	return p.X >= 0 && p.X < pz.W && p.Y >= 0 && p.Y < pz.H
}
