package main

import "sort"

// BFS 전역 재사용 버퍼 — 힙 할당 없이 canReach/isConnected를 실행합니다.
// 솔버는 단일 스레드이므로 안전합니다.
const maxGridCells = 256 // 최대 16×16

var (
	bfsQ   [maxGridCells]Point
	bfsVis [maxGridCells]int
	bfsGen int
)

func Solve(initial *State) (*State, int) {
	calls := 0
	result := solve(initial.Clone(), &calls)
	return result, calls
}

func solve(s *State, calls *int) *State {
	*calls++

	if !applyForcedMoves(s) {
		return nil
	}
	if s.IsSolved() {
		return s
	}
	if !isConnected(s) {
		return nil
	}
	if !parityCheck(s) {
		return nil
	}
	for _, c := range s.Puzzle.Colors {
		if !s.Done[c] && !canReach(s, c) {
			return nil
		}
	}

	c, moves := pickColor(s)
	if c == Empty || len(moves) == 0 {
		return nil
	}

	target := s.Targets[c]
	sort.Slice(moves, func(i, j int) bool {
		return manhattan(moves[i], target) < manhattan(moves[j], target)
	})

	for _, p := range moves {
		next := s.Clone()
		next.Move(c, p)
		if result := solve(next, calls); result != nil {
			return result
		}
	}
	return nil
}

func applyForcedMoves(s *State) bool {
	changed := true
	for changed {
		changed = false

		for _, c := range s.Puzzle.Colors {
			if s.Done[c] {
				continue
			}
			moves := validMoves(s, c)
			switch len(moves) {
			case 0:
				return false
			case 1:
				s.Move(c, moves[0])
				changed = true
			}
		}

		pz := s.Puzzle
		for y := 0; y < pz.H; y++ {
			for x := 0; x < pz.W; x++ {
				if s.Grid[y][x] != Empty {
					continue
				}
				p := Point{x, y}
				emptyN := 0
				headN := 0
				targetN := 0
				var onlyHead Color

				for _, d := range Dirs {
					np := p.Add(d)
					if !pz.InBounds(np) {
						continue
					}
					if s.Grid[np.Y][np.X] == Empty {
						emptyN++
						continue
					}
					for _, c := range pz.Colors {
						if s.Done[c] {
							continue
						}
						if s.Heads[c] == np && s.CanMove(c, p) {
							headN++
							onlyHead = c
						} else if s.Targets[c] == np {
							targetN++
						}
					}
				}

				switch {
				case emptyN == 0 && headN == 0:
					return false
				case emptyN == 0 && headN == 1:
					s.Move(onlyHead, p)
					changed = true
				case emptyN == 1 && headN == 0 && targetN == 0:
					return false
				}
			}
		}
	}
	return true
}

func validMoves(s *State, c Color) []Point {
	head := s.Heads[c]
	var moves []Point
	for _, d := range Dirs {
		p := head.Add(d)
		if s.CanMove(c, p) {
			moves = append(moves, p)
		}
	}
	return moves
}

func pickColor(s *State) (Color, []Point) {
	var best Color
	var bestMoves []Point
	for _, c := range s.Puzzle.Colors {
		if s.Done[c] {
			continue
		}
		moves := validMoves(s, c)
		if best == Empty || len(moves) < len(bestMoves) {
			best = c
			bestMoves = moves
		}
	}
	return best, bestMoves
}

// canReach는 색상 c의 head에서 target까지 빈 칸만 통해 도달 가능한지 BFS로 확인합니다.
// 전역 버퍼를 재사용하므로 할당이 없습니다.
func canReach(s *State, c Color) bool {
	pz := s.Puzzle
	start := s.Heads[c]
	target := s.Targets[c]
	w := pz.W

	bfsGen++
	gen := bfsGen
	bfsVis[start.Y*w+start.X] = gen

	head, tail := 0, 0
	bfsQ[tail] = start
	tail++

	for head < tail {
		cur := bfsQ[head]
		head++
		for _, d := range Dirs {
			np := cur.Add(d)
			if !pz.InBounds(np) {
				continue
			}
			idx := np.Y*w + np.X
			if bfsVis[idx] == gen {
				continue
			}
			if np == target {
				return true
			}
			if s.Grid[np.Y][np.X] == Empty {
				bfsVis[idx] = gen
				bfsQ[tail] = np
				tail++
			}
		}
	}
	return false
}

func manhattan(a, b Point) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// parityCheck는 체스판 격자 채색으로 전체 패리티 제약을 확인합니다.
func parityCheck(s *State) bool {
	pz := s.Puzzle
	nb, nw := 0, 0
	for y := 0; y < pz.H; y++ {
		for x := 0; x < pz.W; x++ {
			if s.Grid[y][x] == Empty {
				if (x+y)%2 == 0 {
					nb++
				} else {
					nw++
				}
			}
		}
	}

	delta := 0
	for _, c := range pz.Colors {
		if s.Done[c] {
			continue
		}
		hp := (s.Heads[c].X + s.Heads[c].Y) % 2
		tp := (s.Targets[c].X + s.Targets[c].Y) % 2
		switch {
		case hp == 0 && tp == 0:
			delta--
		case hp == 1 && tp == 1:
			delta++
		}
	}

	return delta == nb-nw
}

// isConnected는 BFS로 모든 빈 칸이 연결되어 있는지 확인합니다.
// 미완성 색상의 head 위치도 통로로 포함합니다.
// 전역 버퍼를 재사용하므로 할당이 없습니다.
func isConnected(s *State) bool {
	pz := s.Puzzle
	emptyCount := pz.W*pz.H - s.Filled
	if emptyCount == 0 {
		return true
	}

	w := pz.W

	// head 위치를 통로로 마킹
	bfsGen++
	headGen := bfsGen
	for _, c := range pz.Colors {
		if !s.Done[c] {
			h := s.Heads[c]
			bfsVis[h.Y*w+h.X] = headGen
		}
	}

	// 첫 번째 빈 칸 찾기
	var startX, startY int
	found := false
outer:
	for y := 0; y < pz.H; y++ {
		for x := 0; x < pz.W; x++ {
			if s.Grid[y][x] == Empty {
				startX, startY = x, y
				found = true
				break outer
			}
		}
	}
	if !found {
		return true
	}

	bfsGen++
	gen := bfsGen
	startIdx := startY*w + startX
	bfsVis[startIdx] = gen
	head, tail := 0, 0
	bfsQ[head] = Point{startX, startY}
	tail++
	emptyReached := 1

	for head < tail {
		cur := bfsQ[head]
		head++
		for _, d := range Dirs {
			np := cur.Add(d)
			if !pz.InBounds(np) {
				continue
			}
			idx := np.Y*w + np.X
			if bfsVis[idx] == gen {
				continue
			}
			isOpen := s.Grid[np.Y][np.X] == Empty || bfsVis[idx] == headGen
			if !isOpen {
				continue
			}
			bfsVis[idx] = gen
			if s.Grid[np.Y][np.X] == Empty {
				emptyReached++
			}
			bfsQ[tail] = np
			tail++
		}
	}

	return emptyReached == emptyCount
}
