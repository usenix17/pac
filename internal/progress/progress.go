// Package progress parses backend progress output (flatpak) and renders an
// ILoveCandy-style Pac-Man progress bar to match pacman's own bar.
package progress

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var pctRe = regexp.MustCompile(`(\d{1,3})%`)

// Parse returns the last in-range (0-100) percentage found in line, or
// (0,false) if none.
func Parse(line string) (int, bool) {
	matches := pctRe.FindAllStringSubmatch(line, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		n, err := strconv.Atoi(matches[i][1])
		if err == nil && n >= 0 && n <= 100 {
			return n, true
		}
	}
	return 0, false
}

// Bar renders an ILoveCandy-style Pac-Man bar with an inner track of the given
// width for percent (clamped to 0-100): a '-' trail Pac-Man has eaten, a
// chomping head at the frontier ('C' open mouth on even cells, 'c' closed on
// odd, so successive frames read C, c, C, c...), then 'o' pellets on a spaced
// lattice ahead, and the percentage, e.g. "[----C o o o] 40%".
func Bar(percent, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	if width < 3 {
		width = 3
	}
	eaten := percent * width / 100
	head := byte('C') // open mouth
	if eaten%2 == 1 {
		head = 'c' // closed mouth -- the chomp
	}
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < width; i++ {
		switch {
		case i < eaten:
			b.WriteByte('-') // trail Pac-Man has already eaten
		case i == eaten:
			b.WriteByte(head) // Pac-Man at the frontier
		case (i-eaten)%2 == 0:
			b.WriteByte('o') // pellet on the spaced lattice ahead
		default:
			b.WriteByte(' ') // gap between pellets
		}
	}
	b.WriteByte(']')
	return fmt.Sprintf("%s %d%%", b.String(), percent)
}

// Render reads progress lines from r and draws a candy bar to w, redrawing in
// place (carriage return) only when the parsed percentage changes. A final
// newline is written once any bar was drawn.
func Render(r io.Reader, w io.Writer, width int) {
	sc := bufio.NewScanner(r)
	last := -1
	for sc.Scan() {
		if pct, ok := Parse(sc.Text()); ok && pct != last {
			fmt.Fprintf(w, "\r%s", Bar(pct, width))
			last = pct
		}
	}
	if last >= 0 {
		fmt.Fprint(w, "\n")
	}
}
