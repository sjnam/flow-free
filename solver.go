package main

import "sort"

// Global reusable BFS buffer — runs canReach/isConnected without heap allocation.
// Safe because the solver is single-threaded.
const maxGridCells = 256 // max 16×16

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
	if !noIsolatedEmptyRegion(s) {
		return nil
	}
	if !parityCheck(s) {
		return nil
	}
	if !crossingCheck(s) {
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
	// Primary: prefer tighter cells (fewer free neighbors) to fill corners first.
	// Secondary: prefer cells closer to the target.
	sort.Slice(moves, func(i, j int) bool {
		ni, nj := freeNeighbors(s, moves[i]), freeNeighbors(s, moves[j])
		if ni != nj {
			return ni < nj
		}
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
				case emptyN == 1 && headN == 1 && targetN == 0:
					// p has one empty neighbor (ne) and one adjacent head.
					// If ne's only free neighbor is p, the 2-cell chain {p,ne}
					// is a dead-end reachable only via onlyHead → force the move.
					for _, d := range Dirs {
						np2 := p.Add(d)
						if pz.InBounds(np2) && s.Grid[np2.Y][np2.X] == Empty {
							if freeNeighbors(s, np2) == 1 {
								s.Move(onlyHead, p)
								changed = true
							}
							break
						}
					}
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

// freeNeighbors counts empty cells adjacent to p.
func freeNeighbors(s *State, p Point) int {
	n := 0
	for _, d := range Dirs {
		np := p.Add(d)
		if s.Puzzle.InBounds(np) && s.Grid[np.Y][np.X] == Empty {
			n++
		}
	}
	return n
}

func pickColor(s *State) (Color, []Point) {
	var best Color
	var bestMoves []Point
	bestDist := 0

	for _, c := range s.Puzzle.Colors {
		if s.Done[c] {
			continue
		}
		moves := validMoves(s, c)
		dist := manhattan(s.Heads[c], s.Targets[c])
		// MRV primary; tie-break by longer remaining distance first
		// (route paths with more room needed before short ones).
		if best == Empty ||
			len(moves) < len(bestMoves) ||
			(len(moves) == len(bestMoves) && dist > bestDist) {
			best = c
			bestMoves = moves
			bestDist = dist
		}
	}
	return best, bestMoves
}

// canReach uses BFS to check whether color c's head can reach its target through empty cells.
// Reuses the global buffer, so no allocations.
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

// parityCheck verifies the global parity constraint using a checkerboard coloring.
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

// noIsolatedEmptyRegion returns false if any connected component of pure empty
// cells (heads are NOT treated as passable) has no adjacent incomplete color
// head that can enter it.  This is strictly stronger than isConnected:
// isConnected treats heads as bidirectional bridges, which is optimistic —
// a head can only ENTER an adjacent region, not bridge through it.
func noIsolatedEmptyRegion(s *State) bool {
	pz := s.Puzzle
	w := pz.W

	bfsGen++
	gen := bfsGen

	for y := 0; y < pz.H; y++ {
		for x := 0; x < pz.W; x++ {
			idx := y*w + x
			if s.Grid[y][x] != Empty || bfsVis[idx] == gen {
				continue
			}
			// BFS through pure empty cells only; check for any adjacent head.
			hasHead := false
			bfsVis[idx] = gen
			head, tail := 0, 0
			bfsQ[tail] = Point{x, y}
			tail++
			for head < tail {
				cur := bfsQ[head]
				head++
				for _, d := range Dirs {
					np := cur.Add(d)
					if !pz.InBounds(np) {
						continue
					}
					npIdx := np.Y*w + np.X
					if bfsVis[npIdx] == gen {
						continue
					}
					if s.Grid[np.Y][np.X] == Empty {
						bfsVis[npIdx] = gen
						bfsQ[tail] = np
						tail++
					} else if !hasHead {
						for _, c := range pz.Colors {
							if !s.Done[c] && s.Heads[c] == np && s.CanMove(c, cur) {
								hasHead = true
								break
							}
						}
					}
				}
			}
			if !hasHead {
				return false
			}
		}
	}
	return true
}


// crossingCheck returns false if any row or column has become a bottleneck:
// available cells (empty + already secured by must-cross colors) < colors that
// must route through that row/column.
//
// A color "must cross" row r if its path spans across row r:
//   min(head.Y, target.Y) ≤ r ≤ max(head.Y, target.Y)
// Each such color needs ≥ 1 distinct cell in row r.  A color is "secured" if
// it already has any colored cell in row r (endpoint or path segment).
//
// This fires long before isConnected (which fires only when a row is full),
// catching branches where too many cells in a critical row are consumed by
// colors that don't need to cross it.
func crossingCheck(s *State) bool {
	pz := s.Puzzle

	for r := 0; r < pz.H; r++ {
		emptyInRow := 0
		for x := 0; x < pz.W; x++ {
			if s.Grid[r][x] == Empty {
				emptyInRow++
			}
		}
		mustCross, secured := 0, 0
		for _, c := range pz.Colors {
			if s.Done[c] {
				continue
			}
			minY, maxY := s.Heads[c].Y, s.Targets[c].Y
			if minY > maxY {
				minY, maxY = maxY, minY
			}
			if minY > r || maxY < r {
				continue
			}
			mustCross++
			for x := 0; x < pz.W; x++ {
				if s.Grid[r][x] == c {
					secured++
					break
				}
			}
		}
		if emptyInRow+secured < mustCross {
			return false
		}
	}

	for col := 0; col < pz.W; col++ {
		emptyInCol := 0
		for y := 0; y < pz.H; y++ {
			if s.Grid[y][col] == Empty {
				emptyInCol++
			}
		}
		mustCross, secured := 0, 0
		for _, c := range pz.Colors {
			if s.Done[c] {
				continue
			}
			minX, maxX := s.Heads[c].X, s.Targets[c].X
			if minX > maxX {
				minX, maxX = maxX, minX
			}
			if minX > col || maxX < col {
				continue
			}
			mustCross++
			for y := 0; y < pz.H; y++ {
				if s.Grid[y][col] == c {
					secured++
					break
				}
			}
		}
		if emptyInCol+secured < mustCross {
			return false
		}
	}

	return true
}

// isConnected uses BFS to check whether all empty cells are connected.
// Head positions of incomplete colors are also treated as passable.
// Reuses the global buffer, so no allocations.
func isConnected(s *State) bool {
	pz := s.Puzzle
	emptyCount := pz.W*pz.H - s.Filled
	if emptyCount == 0 {
		return true
	}

	w := pz.W

	// Mark head positions as passable
	bfsGen++
	headGen := bfsGen
	for _, c := range pz.Colors {
		if !s.Done[c] {
			h := s.Heads[c]
			bfsVis[h.Y*w+h.X] = headGen
		}
	}

	// Find the first empty cell
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
