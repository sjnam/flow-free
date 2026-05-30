package main

import "math/bits"

// SolveCSP solves the puzzle with degree-constrained CSP + AC-3 propagation.
//
// Encoding:
//   Variables : one Color per cell
//   Constraint: for each cell assigned color c,
//               count of c-colored neighbors must equal
//               1  if the cell is an endpoint of c
//               2  if the cell is an interior path cell
//
// This degree constraint completely characterises valid flow paths: a set of
// cells where every interior node has degree 2 and both endpoints have
// degree 1 is exactly a simple path between the two endpoints.
//
// Propagation (AC-3 style):
//   After assigning cell P to color c:
//     nc  = c-colored neighbors already assigned
//     pc  = unassigned neighbors with c still in their domain
//     req = 1 (endpoint) or 2 (interior)
//     nc > req              → contradiction
//     nc+pc < req           → contradiction
//     nc == req             → remove c from all pc neighbors
//     nc+pc == req          → force all pc neighbors to c
//   When color c is removed from an unassigned cell's domain:
//     re-check constraint of every already-assigned c-neighbor
func SolveCSP(pz *Puzzle) (*State, int) {
	cs := newCSP(pz)
	if !cs.setup() {
		return nil, 0
	}
	calls := 0
	if cspSearch(cs, &calls) {
		return cs.toState(), calls
	}
	return nil, calls
}

// ─── data types ──────────────────────────────────────────────────────────────

type cspCell struct {
	color  Color
	domain uint32 // bit c = color c is in the domain
}

type cspState struct {
	pz    *Puzzle
	cells []cspCell // flat [y*W + x]
}

func newCSP(pz *Puzzle) *cspState {
	full := uint32(0)
	for _, c := range pz.Colors {
		full |= 1 << uint(c)
	}
	cells := make([]cspCell, pz.H*pz.W)
	for i := range cells {
		cells[i].domain = full
	}
	return &cspState{pz: pz, cells: cells}
}

func (cs *cspState) clone() *cspState {
	c2 := make([]cspCell, len(cs.cells))
	copy(c2, cs.cells)
	return &cspState{pz: cs.pz, cells: c2}
}

func (cs *cspState) idx(y, x int) int { return y*cs.pz.W + x }

func (cs *cspState) isEndpoint(y, x int, c Color) bool {
	ep := cs.pz.Endpoints[c]
	return ep[0] == (Point{x, y}) || ep[1] == (Point{x, y})
}

func (cs *cspState) req(y, x int, c Color) int {
	if cs.isEndpoint(y, x, c) {
		return 1
	}
	return 2
}

// ─── constraint propagation ──────────────────────────────────────────────────

// setColor assigns color c to cell (y,x) and propagates degree constraints.
func (cs *cspState) setColor(y, x int, c Color) bool {
	i := cs.idx(y, x)
	cell := &cs.cells[i]
	if cell.color != Empty {
		return cell.color == c
	}
	if (cell.domain>>uint(c))&1 == 0 {
		return false
	}
	oldDomain := cell.domain
	cell.color = c
	cell.domain = 1 << uint(c)

	// For each color d ≠ c that was in the domain, (y,x) is no longer a potential
	// d-neighbor for its already-assigned d-colored neighbors.
	for d := Color(1); d <= Color(len(cs.pz.Colors)+1); d++ {
		if d == c || (oldDomain>>uint(d))&1 == 0 {
			continue
		}
		for _, dir := range Dirs {
			np := Point{x, y}.Add(dir)
			if !cs.pz.InBounds(np) {
				continue
			}
			ni := cs.idx(np.Y, np.X)
			if cs.cells[ni].color == d {
				if !cs.checkCell(np.Y, np.X, d) {
					return false
				}
			}
		}
	}
	return cs.checkCell(y, x, c)
}

// removeDomain removes color c from the domain of unassigned cell (y,x) and propagates.
func (cs *cspState) removeDomain(y, x int, c Color) bool {
	i := cs.idx(y, x)
	cell := &cs.cells[i]
	if cell.color != Empty {
		return cell.color != c
	}
	if (cell.domain>>uint(c))&1 == 0 {
		return true
	}
	cell.domain &^= 1 << uint(c)
	if cell.domain == 0 {
		return false
	}
	// Singleton → force assignment
	if cell.domain&(cell.domain-1) == 0 {
		fc := Color(bits.TrailingZeros32(cell.domain))
		return cs.setColor(y, x, fc)
	}
	// Re-check already-assigned c-neighbors of (y,x) (their pc just decreased).
	for _, dir := range Dirs {
		np := Point{x, y}.Add(dir)
		if !cs.pz.InBounds(np) {
			continue
		}
		ni := cs.idx(np.Y, np.X)
		if cs.cells[ni].color == c {
			if !cs.checkCell(np.Y, np.X, c) {
				return false
			}
		}
	}
	return true
}

// checkCell enforces the degree constraint for cell (y,x) assigned color c.
func (cs *cspState) checkCell(y, x int, c Color) bool {
	if cs.cells[cs.idx(y, x)].color != c {
		return true // stale
	}
	nc, pc := 0, 0
	for _, dir := range Dirs {
		np := Point{x, y}.Add(dir)
		if !cs.pz.InBounds(np) {
			continue
		}
		ni := cs.idx(np.Y, np.X)
		switch {
		case cs.cells[ni].color == c:
			nc++
		case cs.cells[ni].color == Empty && (cs.cells[ni].domain>>uint(c))&1 != 0:
			pc++
		}
	}
	r := cs.req(y, x, c)
	if nc > r {
		return false
	}
	if nc+pc < r {
		return false
	}
	if nc == r {
		// Degree satisfied — remove c from remaining potential neighbors.
		for _, dir := range Dirs {
			np := Point{x, y}.Add(dir)
			if !cs.pz.InBounds(np) {
				continue
			}
			ni := cs.idx(np.Y, np.X)
			if cs.cells[ni].color == Empty && (cs.cells[ni].domain>>uint(c))&1 != 0 {
				if !cs.removeDomain(np.Y, np.X, c) {
					return false
				}
			}
		}
	} else if nc+pc == r {
		// Only possible assignment — force all potential neighbors to c.
		for _, dir := range Dirs {
			np := Point{x, y}.Add(dir)
			if !cs.pz.InBounds(np) {
				continue
			}
			ni := cs.idx(np.Y, np.X)
			if cs.cells[ni].color == Empty && (cs.cells[ni].domain>>uint(c))&1 != 0 {
				if !cs.setColor(np.Y, np.X, c) {
					return false
				}
			}
		}
	}
	return true
}

// setup assigns all endpoint cells and runs initial propagation.
func (cs *cspState) setup() bool {
	for c, ep := range cs.pz.Endpoints {
		for _, p := range ep {
			if !cs.setColor(p.Y, p.X, c) {
				return false
			}
		}
	}
	return true
}

// ─── search ──────────────────────────────────────────────────────────────────

// mrvCell returns the index of the unassigned cell with the smallest domain.
// Returns -1 if all cells are assigned. Returns -2 if a domain is empty.
func (cs *cspState) mrvCell() int {
	best, bestN := -1, 1<<30
	for i, cell := range cs.cells {
		if cell.color != Empty {
			continue
		}
		n := bits.OnesCount32(cell.domain)
		if n == 0 {
			return -2
		}
		if n < bestN {
			bestN, best = n, i
		}
	}
	return best
}

// cspReachable checks that every cell assigned color c is reachable from at
// least one of c's two endpoints via cells that are either already color c or
// still have c in their domain.  An assigned c-cell that cannot be reached
// this way is "orphaned" and can never be part of the valid path → prune.
func (cs *cspState) cspReachable() bool {
	pz := cs.pz
	for _, c := range pz.Colors {
		ep := pz.Endpoints[c]

		bfsGen++
		gen := bfsGen
		head, tail := 0, 0

		enqueue := func(p Point) {
			idx := p.Y*pz.W + p.X
			if bfsVis[idx] != gen {
				bfsVis[idx] = gen
				bfsQ[tail] = p
				tail++
			}
		}
		enqueue(ep[0])
		enqueue(ep[1])

		for head < tail {
			cur := bfsQ[head]
			head++
			for _, d := range Dirs {
				np := cur.Add(d)
				if !pz.InBounds(np) {
					continue
				}
				ni := cs.idx(np.Y, np.X)
				cell := &cs.cells[ni]
				if cell.color == c || (cell.color == Empty && (cell.domain>>uint(c))&1 != 0) {
					enqueue(np)
				}
			}
		}

		// All assigned c-cells must have been reached.
		for y := 0; y < pz.H; y++ {
			for x := 0; x < pz.W; x++ {
				if cs.cells[cs.idx(y, x)].color == c {
					if bfsVis[y*pz.W+x] != gen {
						return false
					}
				}
			}
		}
	}
	return true
}

// cspCrossingCheck returns false if any row or column has too few available
// cells for the colors whose fixed endpoints span it.
func (cs *cspState) cspCrossingCheck() bool {
	pz := cs.pz

	for r := 0; r < pz.H; r++ {
		empty := 0
		for x := 0; x < pz.W; x++ {
			if cs.cells[cs.idx(r, x)].color == Empty {
				empty++
			}
		}
		mustCross, secured := 0, 0
		for c, ep := range pz.Endpoints {
			minY, maxY := ep[0].Y, ep[1].Y
			if minY > maxY {
				minY, maxY = maxY, minY
			}
			if minY > r || maxY < r {
				continue
			}
			mustCross++
			for x := 0; x < pz.W; x++ {
				if cs.cells[cs.idx(r, x)].color == c {
					secured++
					break
				}
			}
		}
		if empty+secured < mustCross {
			return false
		}
	}

	for col := 0; col < pz.W; col++ {
		empty := 0
		for y := 0; y < pz.H; y++ {
			if cs.cells[cs.idx(y, col)].color == Empty {
				empty++
			}
		}
		mustCross, secured := 0, 0
		for c, ep := range pz.Endpoints {
			minX, maxX := ep[0].X, ep[1].X
			if minX > maxX {
				minX, maxX = maxX, minX
			}
			if minX > col || maxX < col {
				continue
			}
			mustCross++
			for y := 0; y < pz.H; y++ {
				if cs.cells[cs.idx(y, col)].color == c {
					secured++
					break
				}
			}
		}
		if empty+secured < mustCross {
			return false
		}
	}

	return true
}

func cspSearch(cs *cspState, calls *int) bool {
	*calls++
	if !cs.cspCrossingCheck() {
		return false
	}
	if !cs.cspReachable() {
		return false
	}
	best := cs.mrvCell()
	if best == -1 {
		return true // all cells assigned
	}
	if best == -2 {
		return false // empty domain
	}
	w := cs.pz.W
	y, x := best/w, best%w
	p := Point{x, y}

	// Value ordering: try colors whose endpoint is closest to this cell first.
	// Closer endpoint ≈ higher probability the path naturally covers this cell.
	type cval struct {
		c    Color
		dist int
	}
	var vals [MaxColors + 1]cval
	nvals := 0
	for bit := cs.cells[best].domain; bit != 0; {
		lsb := bit & (^bit + 1)
		bit &^= lsb
		c := Color(bits.TrailingZeros32(lsb))
		ep := cs.pz.Endpoints[c]
		d0 := manhattan(ep[0], p)
		d1 := manhattan(ep[1], p)
		if d0 > d1 {
			d0 = d1
		}
		vals[nvals] = cval{c, d0}
		nvals++
	}
	// Insertion sort by distance ascending (closest first)
	for i := 1; i < nvals; i++ {
		for j := i; j > 0 && vals[j].dist < vals[j-1].dist; j-- {
			vals[j], vals[j-1] = vals[j-1], vals[j]
		}
	}

	for i := 0; i < nvals; i++ {
		clone := cs.clone()
		if clone.setColor(y, x, vals[i].c) {
			if cspSearch(clone, calls) {
				cs.cells = clone.cells
				return true
			}
		}
	}
	return false
}

// ─── result conversion ───────────────────────────────────────────────────────

func (cs *cspState) toState() *State {
	pz := cs.pz
	state := NewState(pz)
	for y := 0; y < pz.H; y++ {
		for x := 0; x < pz.W; x++ {
			c := cs.cells[cs.idx(y, x)].color
			state.Grid[y][x] = c
		}
	}
	state.Filled = pz.H * pz.W
	for _, c := range pz.Colors {
		state.Done[c] = true
		state.Heads[c] = pz.Endpoints[c][1] // head ends at target
	}
	return state
}
