package main

// MaxColors는 지원하는 최대 색상 수입니다.
const MaxColors = 16

// State는 풀이 과정의 현재 상태입니다.
type State struct {
	Puzzle  *Puzzle
	Grid    [][]Color
	Heads   [MaxColors + 1]Point // Color(1..K) 인덱스 배열
	Targets [MaxColors + 1]Point
	Done    [MaxColors + 1]bool
	Filled  int
}

// NewState는 퍼즐의 초기 상태를 만듭니다.
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

// Clone은 상태를 깊은 복사합니다 (백트래킹용).
// map 대신 배열을 사용하므로 구조체 복사만으로 완성됩니다.
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
		Heads:   s.Heads,   // 배열 값 복사 (힙 할당 없음)
		Targets: s.Targets,
		Done:    s.Done,
		Filled:  s.Filled,
	}
}

// CanMove는 색상 c의 현재 head를 p로 이동할 수 있는지 확인합니다.
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

// Move는 색상 c의 head를 p로 이동시킵니다.
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

// IsSolved는 모든 색상이 연결되고 격자가 꽉 찼는지 확인합니다.
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
