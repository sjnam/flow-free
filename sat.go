package main

import "github.com/crillab/gophersat/solver"

// SolveSAT solves the puzzle by reducing it to SAT and calling a CDCL solver
// (gophersat).
//
// Encoding (the CNF form of the CSP's degree constraint):
//   - One boolean variable per (cell, color): "this cell has this color".
//   - Each cell has exactly one color (the grid must be fully filled).
//   - Endpoint cells are fixed to their color.
//   - Degree: if a cell is color c then it has exactly r same-colored
//     neighbors, where r = 1 for an endpoint of c and 2 otherwise. This
//     exactly characterises a set of simple paths between endpoint pairs.
//
// The degree encoding alone permits a color to also form a detached closed
// loop (all cells degree 2). Such loops are removed lazily: after each model we
// check every color's cells for components not connected to its endpoints and,
// for each, add a clause forbidding that exact set of cells from all being the
// color, then re-solve. The returned count is the number of solve rounds.
func SolveSAT(pz *Puzzle) (*State, int) {
	e := newSatEnc(pz)
	clauses := e.base()
	nbVars := pz.H * pz.W * e.K

	rounds := 0
	for {
		rounds++
		s := solver.New(solver.ParseSliceNb(clauses, nbVars))
		if s.Solve() != solver.Sat {
			return nil, rounds
		}
		grid := e.decode(s.Model())
		blocks := e.cycleBlocks(grid)
		if len(blocks) == 0 {
			return gridToState(pz, grid), rounds
		}
		clauses = append(clauses, blocks...)
	}
}

// ─── encoder ─────────────────────────────────────────────────────────────────

type satEnc struct {
	pz       *Puzzle
	K        int           // number of colors
	colorIdx map[Color]int // color value → 1..K
	colorAt  []Color       // 1..K → color value
}

func newSatEnc(pz *Puzzle) *satEnc {
	k := len(pz.Colors)
	colorIdx := make(map[Color]int, k)
	colorAt := make([]Color, k+1)
	for i, c := range pz.Colors {
		colorIdx[c] = i + 1
		colorAt[i+1] = c
	}
	return &satEnc{pz: pz, K: k, colorIdx: colorIdx, colorAt: colorAt}
}

// varID returns the 1-based DIMACS variable for (cell, color-index ci∈1..K).
func (e *satEnc) varID(cell, ci int) int { return cell*e.K + ci }

// base builds the fixed clause set (everything except lazy cycle blocks).
func (e *satEnc) base() [][]int {
	pz := e.pz
	w := pz.W
	var cl [][]int

	// (1) Each cell has exactly one color.
	for cell := 0; cell < pz.H*pz.W; cell++ {
		atLeastOne := make([]int, e.K)
		for ci := 1; ci <= e.K; ci++ {
			atLeastOne[ci-1] = e.varID(cell, ci)
		}
		cl = append(cl, atLeastOne)
		for ci := 1; ci <= e.K; ci++ {
			for cj := ci + 1; cj <= e.K; cj++ {
				cl = append(cl, []int{-e.varID(cell, ci), -e.varID(cell, cj)})
			}
		}
	}

	// (2) Endpoint cells are fixed to their color.
	for _, c := range pz.Colors {
		ci := e.colorIdx[c]
		for _, p := range pz.Endpoints[c] {
			cl = append(cl, []int{e.varID(p.Y*w+p.X, ci)})
		}
	}

	// (3) Degree constraints.
	for y := 0; y < pz.H; y++ {
		for x := 0; x < pz.W; x++ {
			cell := y*w + x
			var nb []int
			for _, d := range Dirs {
				np := Point{x, y}.Add(d)
				if pz.InBounds(np) {
					nb = append(nb, np.Y*w+np.X)
				}
			}
			for _, c := range pz.Colors {
				ci := e.colorIdx[c]
				s := make([]int, len(nb))
				for k, ncell := range nb {
					s[k] = e.varID(ncell, ci)
				}
				r := 2
				ep := pz.Endpoints[c]
				if ep[0] == (Point{x, y}) || ep[1] == (Point{x, y}) {
					r = 1
				}
				guard := -e.varID(cell, ci) // ¬v: constraint only active when cell is c
				cl = append(cl, clausesAtLeast(guard, s, r)...)
				cl = append(cl, clausesAtMost(guard, s, r)...)
			}
		}
	}

	// (4) No color may fill a complete 2x2 block (a property of every
	// well-formed Flow Free puzzle, and a strong search accelerator).
	for y := 0; y+1 < pz.H; y++ {
		for x := 0; x+1 < pz.W; x++ {
			sq := [4]int{y*w + x, y*w + x + 1, (y+1)*w + x, (y+1)*w + x + 1}
			for ci := 1; ci <= e.K; ci++ {
				cl = append(cl, []int{
					-e.varID(sq[0], ci), -e.varID(sq[1], ci),
					-e.varID(sq[2], ci), -e.varID(sq[3], ci),
				})
			}
		}
	}

	return cl
}

// decode reads a SAT model into a color grid.
func (e *satEnc) decode(model []bool) [][]Color {
	pz := e.pz
	grid := make([][]Color, pz.H)
	for y := 0; y < pz.H; y++ {
		grid[y] = make([]Color, pz.W)
		for x := 0; x < pz.W; x++ {
			cell := y*pz.W + x
			for ci := 1; ci <= e.K; ci++ {
				if model[e.varID(cell, ci)-1] {
					grid[y][x] = e.colorAt[ci]
					break
				}
			}
		}
	}
	return grid
}

// cycleBlocks returns blocking clauses for any color cells that form a
// component detached from that color's endpoints (a closed loop). Returns nil
// when the grid is a clean set of paths.
func (e *satEnc) cycleBlocks(grid [][]Color) [][]int {
	pz := e.pz
	w := pz.W
	var blocks [][]int

	for _, c := range pz.Colors {
		ci := e.colorIdx[c]

		// Cells reachable from endpoint[0] through c-cells (the real path).
		onPath := make([]bool, pz.H*w)
		bfs(pz, pz.Endpoints[c][0], func(p Point) bool { return grid[p.Y][p.X] == c }, onPath)

		// Group the remaining c-cells into components; block each.
		inCycle := make([]bool, pz.H*w)
		for y := 0; y < pz.H; y++ {
			for x := 0; x < pz.W; x++ {
				i := y*w + x
				if grid[y][x] != c || onPath[i] || inCycle[i] {
					continue
				}
				seen := make([]bool, pz.H*w)
				bfs(pz, Point{x, y}, func(p Point) bool { return grid[p.Y][p.X] == c }, seen)
				var clause []int
				for j, s := range seen {
					if s {
						inCycle[j] = true
						clause = append(clause, -e.varID(j, ci))
					}
				}
				blocks = append(blocks, clause)
			}
		}
	}
	return blocks
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// bfs marks in visited every cell reachable from start through cells satisfying
// pass (start is assumed to satisfy it).
func bfs(pz *Puzzle, start Point, pass func(Point) bool, visited []bool) {
	w := pz.W
	visited[start.Y*w+start.X] = true
	q := []Point{start}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, d := range Dirs {
			np := cur.Add(d)
			if !pz.InBounds(np) {
				continue
			}
			idx := np.Y*w + np.X
			if visited[idx] || !pass(np) {
				continue
			}
			visited[idx] = true
			q = append(q, np)
		}
	}
}

// clausesAtLeast encodes "guard ∨ (at least r of s)" as CNF.
// "at least r of n" = every (n-r+1)-subset has a true literal.
func clausesAtLeast(guard int, s []int, r int) [][]int {
	if r <= 0 {
		return nil
	}
	if r > len(s) {
		return [][]int{{guard}} // unreachable degree → cell cannot be this color
	}
	var res [][]int
	for _, sub := range combinations(s, len(s)-r+1) {
		res = append(res, append([]int{guard}, sub...))
	}
	return res
}

// clausesAtMost encodes "guard ∨ (at most r of s)" as CNF.
// "at most r of n" = every (r+1)-subset has a false literal.
func clausesAtMost(guard int, s []int, r int) [][]int {
	if r >= len(s) {
		return nil
	}
	var res [][]int
	for _, sub := range combinations(s, r+1) {
		clause := []int{guard}
		for _, lit := range sub {
			clause = append(clause, -lit)
		}
		res = append(res, clause)
	}
	return res
}

// combinations returns all k-element subsets of items.
func combinations(items []int, k int) [][]int {
	n := len(items)
	if k <= 0 || k > n {
		return nil
	}
	idx := make([]int, k)
	for i := range idx {
		idx[i] = i
	}
	var res [][]int
	for {
		comb := make([]int, k)
		for i, v := range idx {
			comb[i] = items[v]
		}
		res = append(res, comb)

		i := k - 1
		for i >= 0 && idx[i] == n-k+i {
			i--
		}
		if i < 0 {
			break
		}
		idx[i]++
		for j := i + 1; j < k; j++ {
			idx[j] = idx[j-1] + 1
		}
	}
	return res
}

// gridToState wraps a solved color grid in a State for display.
func gridToState(pz *Puzzle, grid [][]Color) *State {
	st := NewState(pz)
	for y := 0; y < pz.H; y++ {
		copy(st.Grid[y], grid[y])
	}
	st.Filled = pz.W * pz.H
	return st
}
