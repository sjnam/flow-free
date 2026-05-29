package main

// Color는 격자 셀의 색상을 나타냅니다. 0은 빈칸입니다.
type Color int

const Empty Color = 0

// Point는 격자 위의 좌표입니다.
type Point struct {
	X, Y int
}

func (p Point) Add(d Point) Point {
	return Point{p.X + d.X, p.Y + d.Y}
}

// 상하좌우 이동 방향
var Dirs = []Point{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

// Puzzle은 문제 원본 데이터입니다. 풀이 중 변경되지 않습니다.
type Puzzle struct {
	W, H       int
	Endpoints  map[Color][2]Point
	Colors     []Color
	ColorNames map[Color]string // 파싱된 경우 원래 문자 (예: "R", "G")
}

// NewPuzzle은 2D 배열로부터 Puzzle을 생성합니다.
// grid[y][x] 값이 0이면 빈칸, 양수면 색상 번호입니다.
// names는 선택적으로 색상 번호 → 표시 문자 매핑을 제공합니다.
func NewPuzzle(grid [][]Color, names ...map[Color]string) *Puzzle {
	h := len(grid)
	w := len(grid[0])
	seen := map[Color]int{}
	endpoints := map[Color][2]Point{}
	var colors []Color

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := grid[y][x]
			if c == Empty {
				continue
			}
			p := Point{x, y}
			pair := endpoints[c]
			if seen[c] == 0 {
				pair[0] = p
				colors = append(colors, c)
			} else {
				pair[1] = p
			}
			endpoints[c] = pair
			seen[c]++
		}
	}

	pz := &Puzzle{
		W:         w,
		H:         h,
		Endpoints: endpoints,
		Colors:    colors,
	}
	if len(names) > 0 {
		pz.ColorNames = names[0]
	}
	return pz
}

// InBounds는 좌표가 격자 범위 안에 있는지 확인합니다.
func (pz *Puzzle) InBounds(p Point) bool {
	return p.X >= 0 && p.X < pz.W && p.Y >= 0 && p.Y < pz.H
}
