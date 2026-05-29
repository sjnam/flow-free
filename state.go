package main

// MaxColors is the maximum number of supported colors.
const MaxColors = 16

// State represents the current state during solving.
type State struct {
	Puzzle  *Puzzle
	Grid    [][]Color
	Heads   [MaxColors + 1]Point // indexed by Color(1..K)
	Targets [MaxColors + 1]Point
	Done    [MaxColors + 1]bool
	Filled  int
}

// NewState creates the initial state for a puzzle.
func NewState(pz *Puzzle) *State {
	h, w := pz.H, pz.W
	flat := make([]Color, h*w)
	grid := make([][]Color, h)
	for y := range grid {
		grid[y] = flat[y*w : (y+1)*w]
	}

	var heads [MaxColors + 1]Point
	var targets [MaxColors + 1]Point
	filled := 0

	for c, pair := range pz.Endpoints {
		grid[pair[0].Y][pair[0].X] = c
		grid[pair[1].Y][pair[1].X] = c
		heads[c] = pair[0]
		targets[c] = pair[1]
		filled += 2
	}

	return &State{
		Puzzle:  pz,
		Grid:    grid,
		Heads:   heads,
		Targets: targets,
		Filled:  filled,
	}
}

// Clone deep-copies the state (for backtracking).
// Uses arrays instead of maps, so a struct copy suffices.
func (s *State) Clone() *State {
	h, w := s.Puzzle.H, s.Puzzle.W
	flat := make([]Color, h*w)
	grid := make([][]Color, h)
	for y := range grid {
		grid[y] = flat[y*w : (y+1)*w]
		copy(grid[y], s.Grid[y])
	}
	return &State{
		Puzzle:  s.Puzzle,
		Grid:    grid,
		Heads:   s.Heads,   // array value copy (no heap allocation)
		Targets: s.Targets,
		Done:    s.Done,
		Filled:  s.Filled,
	}
}

// CanMove reports whether color c's head can move to p.
func (s *State) CanMove(c Color, p Point) bool {
	if !s.Puzzle.InBounds(p) {
		return false
	}
	cell := s.Grid[p.Y][p.X]
	if cell == Empty {
		return true
	}
	return p == s.Targets[c] && cell == c
}

// Move advances color c's head to p.
func (s *State) Move(c Color, p Point) {
	if s.Grid[p.Y][p.X] == Empty {
		s.Grid[p.Y][p.X] = c
		s.Filled++
	}
	s.Heads[c] = p
	if p == s.Targets[c] {
		s.Done[c] = true
	}
}

// IsSolved reports whether all colors are connected and the grid is full.
func (s *State) IsSolved() bool {
	if s.Filled != s.Puzzle.W*s.Puzzle.H {
		return false
	}
	for _, c := range s.Puzzle.Colors {
		if !s.Done[c] {
			return false
		}
	}
	return true
}
