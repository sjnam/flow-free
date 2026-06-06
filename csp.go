package main

import "math/bits"

// SolveCSP solves the puzzle with degree-constrained CSP + AC-3 propagation.
//
// Encoding:
//
//	Variables : one Color per cell
//	Constraint: for each cell assigned color c,
//	            count of c-colored neighbors must equal
//	            1  if the cell is an endpoint of c
//	            2  if the cell is an interior path cell
//
// This degree constraint completely characterises valid flow paths: a set of
// cells where every interior node has degree 2 and both endpoints have
// degree 1 is exactly a simple path between the two endpoints.
//
// Propagation (AC-3 style):
//
//	After assigning cell P to color c:
//	  nc  = c-colored neighbors already assigned
//	  pc  = unassigned neighbors with c still in their domain
//	  req = 1 (endpoint) or 2 (interior)
//	  nc > req              → contradiction
//	  nc+pc < req           → contradiction
//	  nc == req             → remove c from all pc neighbors
//	  nc+pc == req          → force all pc neighbors to c
//	When color c is removed from an unassigned cell's domain:
//	  re-check constraint of every already-assigned c-neighbor
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

// trailEntry records a cell's state before a mutation so it can be undone on
// backtracking, avoiding a full clone of the grid at every search node.
type trailEntry struct {
	idx    int
	color  Color
	domain uint32
}

type cspState struct {
	pz    *Puzzle
	cells []cspCell // flat [y*W + x]
	trail []trailEntry

	// Precomputed, immutable lookups (set in newCSP) to keep the hot
	// propagation/pruning loops free of map access and bounds checks.
	neigh   [][]int               // neigh[i] = in-bounds neighbor flat indices of cell i
	epColor []Color               // epColor[i] = color if cell i is an endpoint, else Empty
	ep      [MaxColors + 1][2]int // ep[c] = the two endpoint flat indices of color c

	// Scratch buffers for cspReachable's BFS, reused across calls. vis uses a
	// generation counter so it never needs clearing between runs.
	q   []int // BFS queue
	vis []int // last generation each cell was visited
	gen int   // current BFS generation
}

func newCSP(pz *Puzzle) *cspState {
	full := uint32(0)
	for _, c := range pz.Colors {
		full |= 1 << uint(c)
	}
	n := pz.H * pz.W
	cells := make([]cspCell, n)
	for i := range cells {
		cells[i].domain = full
	}

	neigh := make([][]int, n)
	for y := 0; y < pz.H; y++ {
		for x := 0; x < pz.W; x++ {
			i := y*pz.W + x
			ns := make([]int, 0, 4)
			for _, dir := range Dirs {
				np := Point{x, y}.Add(dir)
				if pz.InBounds(np) {
					ns = append(ns, np.Y*pz.W+np.X)
				}
			}
			neigh[i] = ns
		}
	}

	epColor := make([]Color, n)
	var ep [MaxColors + 1][2]int
	for c, pair := range pz.Endpoints {
		for k, p := range pair {
			i := p.Y*pz.W + p.X
			epColor[i] = c
			ep[c][k] = i
		}
	}

	return &cspState{
		pz:      pz,
		cells:   cells,
		neigh:   neigh,
		epColor: epColor,
		ep:      ep,
		q:       make([]int, n),
		vis:     make([]int, n),
	}
}

// manhattan returns the L1 distance between two points.
func manhattan(a, b Point) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// record saves cell i's current state on the trail before it is mutated.
func (cs *cspState) record(i int) {
	cs.trail = append(cs.trail, trailEntry{i, cs.cells[i].color, cs.cells[i].domain})
}

// trailMark returns a checkpoint to roll back to.
func (cs *cspState) trailMark() int { return len(cs.trail) }

// rollback restores all cells mutated since the given mark, in reverse order.
func (cs *cspState) rollback(mark int) {
	for i := len(cs.trail) - 1; i >= mark; i-- {
		e := cs.trail[i]
		cs.cells[e.idx].color = e.color
		cs.cells[e.idx].domain = e.domain
	}
	cs.trail = cs.trail[:mark]
}

func (cs *cspState) idx(y, x int) int { return y*cs.pz.W + x }

// req is the required c-colored degree of cell i: 1 for an endpoint of c, else 2.
func (cs *cspState) req(i int, c Color) int {
	if cs.epColor[i] == c {
		return 1
	}
	return 2
}

// ─── constraint propagation ──────────────────────────────────────────────────

// setColor assigns color c to cell i and propagates degree constraints.
func (cs *cspState) setColor(i int, c Color) bool {
	cell := &cs.cells[i]
	if cell.color != Empty {
		return cell.color == c
	}
	if (cell.domain>>uint(c))&1 == 0 {
		return false
	}
	oldDomain := cell.domain
	cs.record(i)
	cell.color = c
	cell.domain = 1 << uint(c)

	// For each color d ≠ c that was in the domain, cell i is no longer a potential
	// d-neighbor for its already-assigned d-colored neighbors.
	for d := Color(1); d <= Color(len(cs.pz.Colors)+1); d++ {
		if d == c || (oldDomain>>uint(d))&1 == 0 {
			continue
		}
		for _, ni := range cs.neigh[i] {
			if cs.cells[ni].color == d {
				if !cs.checkCell(ni, d) {
					return false
				}
			}
		}
	}
	if !cs.no2x2(i, c) {
		return false
	}
	return cs.checkCell(i, c)
}

// no2x2 enforces that color c never fills a complete 2x2 block — a property that
// holds for every well-formed Flow Free puzzle and is a major search accelerator.
// For each 2x2 square containing cell i: four c-cells → contradiction; three
// c-cells with the fourth still able to be c → that fourth cell cannot be c.
func (cs *cspState) no2x2(i int, c Color) bool {
	w, h := cs.pz.W, cs.pz.H
	y, x := i/w, i%w
	for _, off := range [4][2]int{{-1, -1}, {-1, 0}, {0, -1}, {0, 0}} {
		ty, tx := y+off[0], x+off[1]
		if ty < 0 || tx < 0 || ty+1 >= h || tx+1 >= w {
			continue
		}
		sq := [4]int{ty*w + tx, ty*w + tx + 1, (ty+1)*w + tx, (ty+1)*w + tx + 1}
		nc, nEmpty, emptyIdx := 0, 0, -1
		for _, si := range sq {
			switch {
			case cs.cells[si].color == c:
				nc++
			case cs.cells[si].color == Empty && (cs.cells[si].domain>>uint(c))&1 != 0:
				nEmpty++
				emptyIdx = si
			}
		}
		if nc == 4 {
			return false
		}
		if nc == 3 && nEmpty == 1 {
			if !cs.removeDomain(emptyIdx, c) {
				return false
			}
		}
	}
	return true
}

// removeDomain removes color c from the domain of unassigned cell i and propagates.
func (cs *cspState) removeDomain(i int, c Color) bool {
	cell := &cs.cells[i]
	if cell.color != Empty {
		return cell.color != c
	}
	if (cell.domain>>uint(c))&1 == 0 {
		return true
	}
	cs.record(i)
	cell.domain &^= 1 << uint(c)
	if cell.domain == 0 {
		return false
	}
	// Singleton → force assignment
	if cell.domain&(cell.domain-1) == 0 {
		fc := Color(bits.TrailingZeros32(cell.domain))
		return cs.setColor(i, fc)
	}
	// Re-check already-assigned c-neighbors of i (their pc just decreased).
	for _, ni := range cs.neigh[i] {
		if cs.cells[ni].color == c {
			if !cs.checkCell(ni, c) {
				return false
			}
		}
	}
	return true
}

// checkCell enforces the degree constraint for cell i assigned color c.
func (cs *cspState) checkCell(i int, c Color) bool {
	if cs.cells[i].color != c {
		return true // stale
	}
	nc, pc := 0, 0
	for _, ni := range cs.neigh[i] {
		switch {
		case cs.cells[ni].color == c:
			nc++
		case cs.cells[ni].color == Empty && (cs.cells[ni].domain>>uint(c))&1 != 0:
			pc++
		}
	}
	r := cs.req(i, c)
	if nc > r {
		return false
	}
	if nc+pc < r {
		return false
	}
	if nc == r {
		// Degree satisfied — remove c from remaining potential neighbors.
		for _, ni := range cs.neigh[i] {
			if cs.cells[ni].color == Empty && (cs.cells[ni].domain>>uint(c))&1 != 0 {
				if !cs.removeDomain(ni, c) {
					return false
				}
			}
		}
	} else if nc+pc == r {
		// Only possible assignment — force all potential neighbors to c.
		for _, ni := range cs.neigh[i] {
			if cs.cells[ni].color == Empty && (cs.cells[ni].domain>>uint(c))&1 != 0 {
				if !cs.setColor(ni, c) {
					return false
				}
			}
		}
	}
	return true
}

// setup assigns all endpoint cells and runs initial propagation.
func (cs *cspState) setup() bool {
	for _, c := range cs.pz.Colors {
		for _, i := range cs.ep[c] {
			if !cs.setColor(i, c) {
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
	for _, c := range cs.pz.Colors {
		cs.gen++
		gen := cs.gen
		head, tail := 0, 0

		for _, ei := range cs.ep[c] {
			if cs.vis[ei] != gen {
				cs.vis[ei] = gen
				cs.q[tail] = ei
				tail++
			}
		}

		for head < tail {
			cur := cs.q[head]
			head++
			for _, ni := range cs.neigh[cur] {
				if cs.vis[ni] == gen {
					continue
				}
				cell := &cs.cells[ni]
				if cell.color == c || (cell.color == Empty && (cell.domain>>uint(c))&1 != 0) {
					cs.vis[ni] = gen
					cs.q[tail] = ni
					tail++
				}
			}
		}

		// All assigned c-cells must have been reached.
		for i := range cs.cells {
			if cs.cells[i].color == c && cs.vis[i] != gen {
				return false
			}
		}
	}
	return true
}

// cspCrossingCheck returns false if any row or column has too few available
// cells for the colors whose fixed endpoints span it.
func (cs *cspState) cspCrossingCheck() bool {
	pz := cs.pz

	w := pz.W
	for r := 0; r < pz.H; r++ {
		base := r * w
		empty := 0
		for x := 0; x < w; x++ {
			if cs.cells[base+x].color == Empty {
				empty++
			}
		}
		mustCross, secured := 0, 0
		for _, c := range pz.Colors {
			minY, maxY := cs.ep[c][0]/w, cs.ep[c][1]/w
			if minY > maxY {
				minY, maxY = maxY, minY
			}
			if minY > r || maxY < r {
				continue
			}
			mustCross++
			for x := 0; x < w; x++ {
				if cs.cells[base+x].color == c {
					secured++
					break
				}
			}
		}
		if empty+secured < mustCross {
			return false
		}
	}

	for col := 0; col < w; col++ {
		empty := 0
		for y := 0; y < pz.H; y++ {
			if cs.cells[y*w+col].color == Empty {
				empty++
			}
		}
		mustCross, secured := 0, 0
		for _, c := range pz.Colors {
			minX, maxX := cs.ep[c][0]%w, cs.ep[c][1]%w
			if minX > maxX {
				minX, maxX = maxX, minX
			}
			if minX > col || maxX < col {
				continue
			}
			mustCross++
			for y := 0; y < pz.H; y++ {
				if cs.cells[y*w+col].color == c {
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
		mark := cs.trailMark()
		if cs.setColor(best, vals[i].c) && cspSearch(cs, calls) {
			return true
		}
		cs.rollback(mark)
	}
	return false
}

// ─── result conversion ───────────────────────────────────────────────────────

func (cs *cspState) toState() *State {
	pz := cs.pz
	state := NewState(pz)
	for y := 0; y < pz.H; y++ {
		for x := 0; x < pz.W; x++ {
			state.Grid[y][x] = cs.cells[cs.idx(y, x)].color
		}
	}
	state.Filled = pz.H * pz.W
	return state
}
