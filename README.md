# Flow Free

A fast solver for [Flow Free](https://en.wikipedia.org/wiki/Flow_Free) puzzles
(the grid-filling pipe-connection game, also known as *Numberlink*), written in Go.

Given a grid with pairs of colored endpoints, the solver finds non-crossing
paths that connect each pair **and fill every cell**.

## Build & run

```sh
go build .
./flow-free puzzle.txt      # solve a puzzle file (CSP solver)
./flow-free -sat puzzle.txt # solve via SAT reduction (gophersat)
./flow-free -               # read a puzzle from stdin
```

Or without building:

```sh
go run . puzzle.txt
```

## Example

Input (`8x8.txt`):

```
. . . . . . . G
. R . . . . . B
. C . . . O . G
. O C R . B . .
. . . M . M . .
. Y . . . . . .
. . . . . . Y .
. . . . . . . .
```

Running `./flow-free 8x8.txt` prints the puzzle, the search stats, and the
solved grid (in the terminal each color is drawn as a colored dot `●`; shown
here as letters):

```
[csp] Calls: 4  Elapsed: 75.958µs

┌────────────────────────┐
│ G  G  G  G  G  G  G  G │
│ G  R  R  R  B  B  B  B │
│ G  C  C  R  B  O  O  G │
│ G  O  C  R  B  B  O  G │
│ G  O  O  M  M  M  O  G │
│ G  Y  O  O  O  O  O  G │
│ G  Y  Y  Y  Y  Y  Y  G │
│ G  G  G  G  G  G  G  G │
└────────────────────────┘
Filled: 64/64
```

`Calls` is the number of search nodes explored — strong pruning keeps it tiny
(e.g. an 11×14 / 8-color puzzle solves in ~140 nodes, a few milliseconds).

## Puzzle format

- One line per row; `#`-prefixed lines are comments.
- Cells: a letter (case-insensitive) marks a color endpoint; `.` or `0` is empty.
- Cells may be space-separated (`R . B`) or packed (`R.B`).
- Each color must appear **exactly twice** (its two endpoints).

## How it works

The solver models the puzzle as a **degree-constrained CSP** and solves it with
constraint propagation plus backtracking search ([csp.go](csp.go)).

**Encoding.** One variable per cell, whose domain is the set of possible colors.
A filled grid is a valid solution iff, for every cell of color `c`, its number of
same-colored orthogonal neighbors equals:

- **1** if the cell is an endpoint of `c`,
- **2** otherwise (an interior path cell).

This degree rule exactly characterizes a set of simple paths between endpoint
pairs.

**Propagation (AC-3 style).** After each assignment the solver enforces the
degree constraint locally and forces/eliminates colors in neighboring cells.
When a cell's domain collapses to one color it is assigned automatically, which
cascades.

**Pruning.** Several checks cut dead branches early:

- **Connectivity** ([`cspReachable`](csp.go)): every cell committed to color `c`
  must remain reachable from one of `c`'s endpoints — this also rules out
  detached loops.
- **Crossing bound** ([`cspCrossingCheck`](csp.go)): a row/column cannot offer
  fewer free cells than the number of colors that must pass through it.
- **No 2×2 block** ([`no2x2`](csp.go)): no color may fill a complete 2×2 square.
  This holds for every well-formed Flow Free puzzle and is the single biggest
  accelerator.

**Search.** Variable selection uses **MRV** (pick the cell with the smallest
domain); value ordering tries the color whose endpoint is nearest first.
Backtracking is **trail-based** — mutations are recorded and undone in place, so
no per-node copy of the grid is allocated.

### SAT solver (`-sat`)

[sat.go](sat.go) offers an alternative: reduce the puzzle to SAT and let a CDCL
solver ([gophersat](https://github.com/crillab/gophersat)) do the work. The
encoding is the CNF form of the same degree constraint — one boolean per
`(cell, color)`, exactly one color per cell, endpoints fixed, and "if a cell is
color `c` it has exactly 1 (endpoint) or 2 (interior) `c`-neighbors". The
no-2×2-block rule is added as clauses too (one per square per color), which
roughly halves solve time on the larger puzzles.

The degree encoding alone allows a color to also form a detached closed loop, so
loops are removed **lazily**: after each model, any color cells not connected to
that color's endpoints are forbidden with a blocking clause and the solver runs
again. `Rounds` reports how many solve passes this took (1 when no loop appears,
which is the common case).

## Project layout

| File | Purpose |
| --- | --- |
| [csp.go](csp.go) | the CSP solver: propagation, pruning, and search |
| [sat.go](sat.go) | the SAT encoding and lazy cycle elimination |
| [puzzle.go](puzzle.go) | `Puzzle` model and endpoint extraction |
| [puzzle_io.go](puzzle_io.go) | parsing puzzle text |
| [state.go](state.go) | solved-grid container |
| [display.go](display.go) | terminal rendering |
| [main.go](main.go) | CLI entry point |

Sample puzzles of various sizes are included (`8x8.txt`, `9x9.txt`,
`10x10.txt`, `11x14.txt`, …).
