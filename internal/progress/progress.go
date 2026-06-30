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

// itemRe matches flatpak's per-operation progress header, e.g.
// "Installing 1/2…" / "Updating 3/5…", capturing the item index and count.
var itemRe = regexp.MustCompile(`(?:Installing|Updating|Uninstalling|Downloading) (\d+)/(\d+)`)

// tableRe matches flatpak's numbered ref table that precedes the transfer, e.g.
// " 1.   <TAB> org.gnome.Calculator.Locale <TAB> stable ...", mapping the item
// index to its ref so we can label that item's bar.
var tableRe = regexp.MustCompile(`^\s*(\d+)\.\s+(\S+)`)

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

// labeled formats one bar line as "<name>  <bar>", padding the name into a
// fixed column so successive items' bars line up (pacman-style). An empty name
// renders the bare bar.
func labeled(name string, pct, width int) string {
	if name == "" {
		return Bar(pct, width)
	}
	return fmt.Sprintf("%-28s %s", clip(name, 28), Bar(pct, width))
}

// clip truncates s to at most n bytes, marking truncation with a trailing "..".
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 2 {
		return s[:n]
	}
	return s[:n-2] + ".."
}

// Render reads flatpak's progress output from r and draws a labeled candy bar
// to w, one line per item: "<ref>  [----C o o o] NN%". It redraws the current
// line in place (carriage return) only when the percentage changes, and starts
// a fresh line when flatpak moves to the next item. label is the caller's known
// target (e.g. the app id for a single install); it is used until the stream
// reveals a per-item ref, and as the fallback when none is found.
//
// Render always consumes r to completion (even on a scan error it drains the
// rest), so the child writing to the pipe never blocks on a full buffer and the
// caller's cmd.Wait cannot deadlock. It returns any scanner error: a failed
// scan (e.g. a line over the token limit) must not be mistaken for clean EOF.
func Render(r io.Reader, w io.Writer, width int, label string) error {
	sc := bufio.NewScanner(r)
	// flatpak can emit long progress/table lines; grow the max token to 1 MiB
	// so an ordinary line never trips bufio.Scanner's default 64 KiB limit.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	refs := map[int]string{} // item index -> ref, learned from the table
	cur := label             // label for the item currently drawing
	idx := 0                 // current item index (0 = none yet)
	last := -1               // last percent drawn for this item
	drawn := false           // has the current item's line had any output
	for sc.Scan() {
		line := sc.Text()
		if m := tableRe.FindStringSubmatch(line); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				refs[n] = m[2]
			}
			continue
		}
		if m := itemRe.FindStringSubmatch(line); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n != idx {
				if drawn {
					fmt.Fprint(w, "\n") // finalize the previous item's line
				}
				idx = n
				switch {
				case refs[n] != "":
					cur = refs[n]
				case label != "":
					cur = label
				default:
					cur = m[1] + "/" + m[2]
				}
				last = -1
				drawn = false
			}
		}
		if pct, ok := Parse(line); ok && pct != last {
			fmt.Fprintf(w, "\r%s", labeled(cur, pct, width))
			last = pct
			drawn = true
		}
	}
	if drawn {
		fmt.Fprint(w, "\n")
	}
	if err := sc.Err(); err != nil {
		// Drain whatever the child keeps writing so its stdout pipe never fills
		// and blocks it; only then surface the error.
		_, _ = io.Copy(io.Discard, r)
		return err
	}
	return nil
}
