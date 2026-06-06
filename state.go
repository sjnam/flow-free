package main

// MaxColors is the maximum number of supported colors.
const MaxColors = 16

// State holds a solved (or to-be-displayed) grid.
type State struct {
	Puzzle *Puzzle
	Grid   [][]Color
	Filled int
}

// NewState allocates a State with endpoints placed on the grid.
func NewState(pz *Puzzle) *State {
	h, w := pz.H, pz.W
	flat := make([]Color, h*w)
	grid := make([][]Color, h)
	for y := range grid {
		grid[y] = flat[y*w : (y+1)*w]
	}

	filled := 0
	for c, pair := range pz.Endpoints {
		grid[pair[0].Y][pair[0].X] = c
		grid[pair[1].Y][pair[1].X] = c
		filled += 2
	}

	return &State{Puzzle: pz, Grid: grid, Filled: filled}
}
