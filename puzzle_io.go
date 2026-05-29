package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// ReadPuzzle은 io.Reader에서 퍼즐 텍스트를 읽어 파싱합니다.
func ReadPuzzle(r io.Reader) ([][]Color, map[Color]string, error) {
	var sb strings.Builder
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return ParseGrid(sb.String())
}

// ParseGrid는 텍스트를 파싱해 격자와 색상 이름 매핑을 반환합니다.
//
// 형식:
//   - 한 줄 = 한 행
//   - '#' 로 시작하는 줄은 주석 (무시)
//   - 셀: 알파벳 대소문자 → 색상 endpoint, '.' 또는 '0' → 빈 칸
//   - 셀 구분: 공백 포함("R . B") 또는 붙여쓰기("R.B") 모두 지원
//   - 각 색상은 정확히 2번 등장해야 합니다
func ParseGrid(text string) ([][]Color, map[Color]string, error) {
	lines := strings.Split(strings.TrimSpace(text), "\n")

	var rows []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("입력이 비어 있습니다")
	}

	letterToColor := map[rune]Color{}
	colorNames := map[Color]string{}
	nextColor := Color(1)

	assign := func(ch rune) Color {
		ch = unicode.ToUpper(ch)
		if c, ok := letterToColor[ch]; ok {
			return c
		}
		c := nextColor
		nextColor++
		letterToColor[ch] = c
		colorNames[c] = string(ch)
		return c
	}

	var grid [][]Color
	width := -1

	for y, line := range rows {
		var tokens []string
		if strings.ContainsRune(line, ' ') {
			tokens = strings.Fields(line)
		} else {
			for _, ch := range line {
				tokens = append(tokens, string(ch))
			}
		}

		if width < 0 {
			width = len(tokens)
		} else if len(tokens) != width {
			return nil, nil, fmt.Errorf("%d번 행: 칸 수 불일치 (기대 %d, 실제 %d)", y+1, width, len(tokens))
		}

		var row []Color
		for _, tok := range tokens {
			runes := []rune(tok)
			switch {
			case tok == "." || tok == "0":
				row = append(row, Empty)
			case len(runes) == 1 && unicode.IsLetter(runes[0]):
				row = append(row, assign(runes[0]))
			default:
				return nil, nil, fmt.Errorf("알 수 없는 셀 값: %q", tok)
			}
		}
		grid = append(grid, row)
	}

	// 각 색상이 정확히 2번 등장하는지 검증
	count := map[Color]int{}
	for _, row := range grid {
		for _, c := range row {
			if c != Empty {
				count[c]++
			}
		}
	}
	for c, n := range count {
		if n != 2 {
			return nil, nil, fmt.Errorf("색상 %q: endpoint가 %d개입니다 (정확히 2개여야 합니다)", colorNames[c], n)
		}
	}

	return grid, colorNames, nil
}
