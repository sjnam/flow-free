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
//   - one line = one row; lines starting with '#' are comments (ignored)
//   - cells: letters → color endpoints, '.' or '0' → empty cell,
//     '*' → wall (a hole that is not part of the board)
//   - separators: space-separated ("R . B") or packed ("R.B") are both supported
//   - rows may differ in length; short rows are right-padded with walls, so the
//     playable area can be non-rectangular (e.g. an hourglass)
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

	// First pass: tokenize every row.
	tokenRows := make([][]string, len(rows))
	width := 0
	for y, line := range rows {
		var tokens []string
		if strings.ContainsRune(line, ' ') {
			tokens = strings.Fields(line)
		} else {
			for _, ch := range line {
				tokens = append(tokens, string(ch))
			}
		}
		tokenRows[y] = tokens
		if len(tokens) > width {
			width = len(tokens)
		}
	}

	// Second pass: build the grid, padding short rows with walls.
	grid := make([][]Color, len(tokenRows))
	for y, tokens := range tokenRows {
		row := make([]Color, width)
		for x := range row {
			row[x] = Wall // default for padded cells
		}
		for x, tok := range tokens {
			runes := []rune(tok)
			switch {
			case tok == "." || tok == "0":
				row[x] = Empty
			case tok == "*":
				row[x] = Wall
			case len(runes) == 1 && unicode.IsLetter(runes[0]):
				row[x] = assign(runes[0])
			default:
				return nil, nil, fmt.Errorf("row %d: unknown cell value: %q", y+1, tok)
			}
		}
		grid[y] = row
	}

	// Validate that each color appears exactly twice.
	count := map[Color]int{}
	for _, row := range grid {
		for _, c := range row {
			if c != Empty && c != Wall {
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
