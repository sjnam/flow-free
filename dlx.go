package main

import "slices"

// SolveDLX solves the puzzle with a lazy online exact-cover search.
//
// Unlike the step-by-step backtracking solver that grows all paths one cell
// at a time, this approach commits to one color's *complete* path at a time:
//
//	dlxSolve  → pick the most-constrained remaining color (MRV)
//	dlxBuild  → grow that color's path cell-by-cell via DFS
//	          → when the path reaches its endpoint, recurse into dlxSolve
//
// The key insight (the "lazy" part): we never enumerate all paths upfront.
// Instead, the path generator and the exact-cover search are interleaved —
// each DFS step immediately tests whether the partial path is still viable
// (connectivity, reachability, parity) before going deeper.
//
// This corresponds to Knuth's "Dancing Cells" MRV-branching strategy applied
// to the color-selection level, with the path DFS acting as the option
// generator that produces one option at a time on demand.
func SolveDLX(pz *Puzzle) (*State, int) {
	s := NewState(pz)
	calls := 0
	colors := make([]Color, len(pz.Colors))
	copy(colors, pz.Colors)
	if dlxSolve(s, colors, &calls) {
		return s, calls
	}
	return nil, calls
}

// forcedMove records one step of forced propagation so it can be undone.
type forcedMove struct {
	c        Color
	prevHead Point
	to       Point
}

// dlxApplyForced propagates forced moves for the active color set.
//
// Rule 1 — color with 0 valid moves: infeasible.
// Rule 1 — color with exactly 1 valid move: force that move.
// Rule 2 — empty cell with no accessible head neighbor at all: infeasible.
// Rule 2 — empty cell with exactly one accessible head (from ALL incomplete
//
//	colors) and that head is in active: force the move.
//
// Using all incomplete colors for cell-neighbor counting (not just active) is
// critical: when dlxBuild is growing color c (not in active=rest), c's head
// may be the only color that can reach a cell.  If we only counted active
// colors, we'd incorrectly flag the cell as unreachable and return false.
//
// Returns the list of moves applied and whether the state is still feasible.
// On failure the partial list still contains any moves made before the
// infeasibility was detected — caller must call dlxUndoForced to restore.
func dlxApplyForced(s *State, active []Color) ([]forcedMove, bool) {
	var forced []forcedMove
	pz := s.Puzzle

	changed := true
	for changed {
		changed = false

		// Rule 1: active color with 0 or 1 valid moves.
		for _, c := range active {
			if s.Done[c] {
				continue
			}
			var buf [4]Point
			mvs := buf[:0]
			for _, d := range Dirs {
				if p := s.Heads[c].Add(d); s.CanMove(c, p) {
					mvs = append(mvs, p)
				}
			}
			switch len(mvs) {
			case 0:
				return forced, false
			case 1:
				prev := s.Heads[c]
				s.Move(c, mvs[0])
				forced = append(forced, forcedMove{c, prev, mvs[0]})
				changed = true
			}
		}

		// Rule 2: cell-based forced moves.
		// Count neighbors from ALL incomplete colors so that c's head (not in
		// active during dlxBuild) is visible and cells aren't falsely flagged.
		for y := 0; y < pz.H; y++ {
			for x := 0; x < pz.W; x++ {
				if s.Grid[y][x] != Empty {
					continue
				}
				p := Point{x, y}
				emptyN, totalHeadN, activeHeadN, targetN := 0, 0, 0, 0
				var onlyActiveHead Color
				var onlyActivePrev Point

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
							totalHeadN++
							if slices.Contains(active, c) {
								activeHeadN++
								onlyActiveHead = c
								onlyActivePrev = np
							}
						} else if s.Targets[c] == np {
							targetN++
						}
					}
				}

				switch {
				case emptyN == 0 && totalHeadN == 0:
					return forced, false
				case emptyN == 0 && totalHeadN == 1 && activeHeadN == 1:
					s.Move(onlyActiveHead, p)
					forced = append(forced, forcedMove{onlyActiveHead, onlyActivePrev, p})
					changed = true
				case emptyN == 1 && totalHeadN == 0 && targetN == 0:
					return forced, false
				case emptyN == 1 && totalHeadN == 1 && activeHeadN == 1 && targetN == 0:
					for _, d := range Dirs {
						np2 := p.Add(d)
						if pz.InBounds(np2) && s.Grid[np2.Y][np2.X] == Empty {
							if freeNeighbors(s, np2) == 1 {
								s.Move(onlyActiveHead, p)
								forced = append(forced, forcedMove{onlyActiveHead, onlyActivePrev, p})
								changed = true
							}
							break
						}
					}
				}
			}
		}
	}
	return forced, true
}

// dlxUndoForced reverses the moves recorded by dlxApplyForced in LIFO order.
func dlxUndoForced(s *State, forced []forcedMove) {
	for i := len(forced) - 1; i >= 0; i-- {
		fm := forced[i]
		s.UndoMove(fm.c, fm.prevHead, fm.to)
	}
}

// dlxSolve picks the most-constrained color (MRV: smallest BFS-reachable region)
// and hands off to dlxBuild.  The colors slice is mutated in-place for the swap
// but callers don't depend on its order, so no restore is needed.
func dlxSolve(s *State, colors []Color, calls *int) bool {
	if len(colors) == 0 {
		return s.IsSolved()
	}

	// Propagate forced moves across all remaining colors before choosing.
	forced, ok := dlxApplyForced(s, colors)
	if !ok {
		dlxUndoForced(s, forced)
		return false
	}

	// Global feasibility checks after forced propagation.
	if !parityCheck(s) {
		dlxUndoForced(s, forced)
		return false
	}
	if !noIsolatedEmptyRegion(s) {
		dlxUndoForced(s, forced)
		return false
	}
	if !dlxAllReachable(s, 0, colors) {
		dlxUndoForced(s, forced)
		return false
	}
	for _, c := range colors {
		if !s.Done[c] && !canReach(s, c) {
			dlxUndoForced(s, forced)
			return false
		}
	}

	// MRV: pick the color whose head can reach the fewest cells.
	// Tie-break: longest manhattan distance to target (commit far-flung paths first).
	// In open grids (early stages) all reach counts are equal, so the tie-break
	// routes colors that span large distances first, anchoring the open area.
	best, bestN, bestDist := -1, 1<<30, -1
	for i, c := range colors {
		if s.Done[c] {
			continue
		}
		n := dlxReachCount(s, c)
		d := manhattan(s.Heads[c], s.Targets[c])
		if n < bestN || (n == bestN && d > bestDist) {
			bestN, bestDist, best = n, d, i
		}
	}
	if best == -1 {
		// All remaining colors were forced to completion.
		result := s.IsSolved()
		if !result {
			dlxUndoForced(s, forced)
		}
		return result
	}

	colors[0], colors[best] = colors[best], colors[0]
	result := dlxBuild(s, colors[0], colors[1:], calls)
	if !result {
		dlxUndoForced(s, forced)
	}
	return result
}

// dlxBuild grows color c's path one cell at a time (DFS).
// After each step, forced moves for the rest colors are propagated immediately.
// When Done[c] becomes true the path is complete; it then calls dlxSolve
// to continue with the remaining colors.
// All moves (own + rest-forced) are undone on backtrack via UndoMove/dlxUndoForced.
func dlxBuild(s *State, c Color, rest []Color, calls *int) bool {
	*calls++

	if s.Done[c] {
		return dlxSolve(s, rest, calls)
	}

	head := s.Heads[c]
	target := s.Targets[c]

	// Collect valid next cells (at most 4).
	var buf [4]Point
	moves := buf[:0]
	for _, d := range Dirs {
		if p := head.Add(d); s.CanMove(c, p) {
			moves = append(moves, p)
		}
	}

	// Greedy tie-break: try cells closer to the target first.
	for i := 1; i < len(moves); i++ {
		for j := i; j > 0 && manhattan(moves[j], target) < manhattan(moves[j-1], target); j-- {
			moves[j], moves[j-1] = moves[j-1], moves[j]
		}
	}

	for _, next := range moves {
		s.Move(c, next)

		// Propagate forced moves for rest colors before pruning.
		// Uses all puzzle colors for cell-neighbor counting (see dlxApplyForced).
		forced, ok := dlxApplyForced(s, rest)
		if ok && dlxPrune(s, c, rest) {
			if dlxBuild(s, c, rest, calls) {
				return true
			}
		}
		dlxUndoForced(s, forced)
		s.UndoMove(c, head, next)
	}
	return false
}

// dlxPrune applies feasibility checks after each path step.
//
// Why not isConnected?
// isConnected does a single-source BFS from the first empty cell, treating
// incomplete-color heads as passable bridges.  When one color's growing path
// has divided the grid, the remaining colors' starting endpoints (pair[0])
// may lie in a disconnected region unreachable from the BFS origin — the
// solver then wrongly prunes feasible states.  dlxAllReachable seeds the BFS
// from ALL incomplete-color heads simultaneously (multi-source), so every
// region that contains at least one unstarted color's endpoint is reachable.
func dlxPrune(s *State, c Color, rest []Color) bool {
	if !s.Done[c] && !canReach(s, c) {
		return false
	}
	if !parityCheck(s) {
		return false
	}
	if !noIsolatedEmptyRegion(s) {
		return false
	}
	if !dlxAllReachable(s, c, rest) {
		return false
	}
	for _, rc := range rest {
		if s.Done[rc] {
			continue
		}
		if !canReach(s, rc) {
			return false
		}
	}
	return true
}

// dlxAllReachable runs a multi-source BFS from every incomplete color's head
// and checks that all empty cells are reachable from at least one source.
// Unlike isConnected, heads are treated as starting points (not bridges),
// so regions isolated by the current partial path but containing another
// color's starting endpoint are still correctly flagged as reachable.
func dlxAllReachable(s *State, c Color, rest []Color) bool {
	pz := s.Puzzle
	w := pz.W
	emptyCount := pz.W*pz.H - s.Filled
	if emptyCount == 0 {
		return true
	}

	bfsGen++
	gen := bfsGen
	head, tail, reached := 0, 0, 0

	enqueue := func(p Point) {
		idx := p.Y*w + p.X
		if bfsVis[idx] != gen {
			bfsVis[idx] = gen
			bfsQ[tail] = p
			tail++
		}
	}

	// c==0 means no current color (called from dlxSolve before picking one).
	if c != 0 && !s.Done[c] {
		enqueue(s.Heads[c])
	}
	for _, rc := range rest {
		if !s.Done[rc] {
			enqueue(s.Heads[rc])
		}
	}

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
			if s.Grid[np.Y][np.X] == Empty {
				bfsVis[idx] = gen
				reached++
				bfsQ[tail] = np
				tail++
			}
		}
	}
	return reached == emptyCount
}

// dlxReachCount returns the number of cells reachable from c's current head
// through empty cells (the target endpoint counts as reachable even though
// it is pre-filled).  Used as an MRV proxy: fewer reachable cells ≈ fewer
// viable complete paths.
func dlxReachCount(s *State, c Color) int {
	pz := s.Puzzle
	start := s.Heads[c]
	target := s.Targets[c]
	w := pz.W

	bfsGen++
	gen := bfsGen
	bfsVis[start.Y*w+start.X] = gen
	bfsQ[0] = start
	head, tail, count := 0, 1, 0

	for head < tail {
		cur := bfsQ[head]
		head++
		count++
		for _, d := range Dirs {
			np := cur.Add(d)
			if !pz.InBounds(np) {
				continue
			}
			idx := np.Y*w + np.X
			if bfsVis[idx] == gen {
				continue
			}
			if np == target || s.Grid[np.Y][np.X] == Empty {
				bfsVis[idx] = gen
				bfsQ[tail] = np
				tail++
			}
		}
	}
	return count
}
