package main

import (
	"fmt"
	"os"
	"time"
)

func run(name string, grid [][]Color, names map[Color]string) {
	fmt.Printf("=== %s ===\n", name)
	pz := NewPuzzle(grid, names)
	PrintPuzzle(pz, grid)

	fmt.Println("\n풀이 중...")
	start := time.Now()
	result, calls := Solve(NewState(pz))
	elapsed := time.Since(start)

	fmt.Printf("호출 횟수: %d회  소요 시간: %v\n\n", calls, elapsed)
	if result == nil {
		fmt.Println("풀 수 없습니다.")
		return
	}
	PrintState(result)
	fmt.Println()
}

const usage = `사용법:
  go run . puzzle.txt       파일에서 퍼즐 읽기
  go run . -                표준 입력에서 퍼즐 읽기

퍼즐 형식 (puzzle.txt 예시):
  # 주석은 '#'으로 시작
  R . . . B
  . . . . .
  . G . B .
  . . . . .
  R . G . .

  규칙:
  - 알파벳 대소문자: 색상 endpoint (한 색상당 정확히 2개)
  - '.' 또는 '0': 빈 칸
  - 공백 구분("R . B") 또는 붙여쓰기("R.B") 모두 가능
`

func main() {
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		fmt.Print(usage)
		return
	}

	// 파일 또는 stdin에서 퍼즐 읽기
	var (
		grid  [][]Color
		names map[Color]string
		err   error
	)

	if os.Args[1] == "-" {
		grid, names, err = ReadPuzzle(os.Stdin)
	} else {
		f, openErr := os.Open(os.Args[1])
		if openErr != nil {
			fmt.Fprintf(os.Stderr, "오류: %v\n", openErr)
			os.Exit(1)
		}
		defer f.Close()
		grid, names, err = ReadPuzzle(f)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "파싱 오류: %v\n", err)
		os.Exit(1)
	}

	run(os.Args[1], grid, names)
}
