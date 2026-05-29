package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// ReadPuzzle reads and parses puzzle text from an io.Reader.
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

// ParseGrid parses text and returns a grid and color name mapping.
//
// Format:
//   - one line = one row
//   - lines starting with '#' are comments (ignored)
//   - cells: letters → color endpoints, '.' or '0' → empty cell
//   - separators: space-separated ("R . B") or packed ("R.B") are both supported
//   - each color must appear exactly twice
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
		return nil, nil, fmt.Errorf("empty input")
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
			return nil, nil, fmt.Errorf("row %d: column count mismatch (expected %d, got %d)", y+1, width, len(tokens))
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
				return nil, nil, fmt.Errorf("unknown cell value: %q", tok)
			}
		}
		grid = append(grid, row)
	}

	// Validate that each color appears exactly twice
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
			return nil, nil, fmt.Errorf("color %q: has %d endpoints (must be exactly 2)", colorNames[c], n)
		}
	}

	return grid, colorNames, nil
}
