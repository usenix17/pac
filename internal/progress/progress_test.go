package progress_test

import (
	"strings"
	"testing"

	"starnix.net/pac/internal/progress"
)

func TestParsePercent(t *testing.T) {
	cases := []struct {
		line string
		want int
		ok   bool
	}{
		{"Installing 41% done", 41, true},
		{"100% complete", 100, true},
		{"  7%", 7, true},
		{"no percent here", 0, false},
		{"999% bogus", 0, false}, // out of range -> not ok
		{"downloading 12% then 34%", 34, true}, // last wins
	}
	for _, c := range cases {
		got, ok := progress.Parse(c.line)
		if got != c.want || ok != c.ok {
			t.Errorf("Parse(%q) = (%d,%v), want (%d,%v)", c.line, got, ok, c.want, c.ok)
		}
	}
}

func TestBarFormat(t *testing.T) {
	// 40% of width 11 -> 4 eaten cells, open mouth, spaced pellets ahead.
	if got, want := progress.Bar(40, 11), "[----C o o o] 40%"; got != want {
		t.Fatalf("Bar(40,11) = %q, want %q", got, want)
	}
}

func TestBarChomp(t *testing.T) {
	// As the head advances a cell the mouth alternates open/closed: C, c, C...
	// width 20 -> one eaten cell per 5%.
	cases := []struct {
		percent int
		head    byte
	}{
		{20, 'C'}, // 4 eaten (even) -> open
		{25, 'c'}, // 5 eaten (odd)  -> closed
		{30, 'C'}, // 6 eaten (even) -> open
	}
	for _, c := range cases {
		b := progress.Bar(c.percent, 20)
		if !strings.ContainsRune(b, rune(c.head)) {
			t.Errorf("Bar(%d,20) = %q, want head %q", c.percent, b, c.head)
		}
	}
}

func TestBarClampsAndEnds(t *testing.T) {
	if got := progress.Bar(150, 8); !strings.HasSuffix(got, "100%") {
		t.Fatalf("Bar(150) = %q, want clamped to 100%%", got)
	}
	if got := progress.Bar(-5, 8); !strings.HasSuffix(got, "0%") {
		t.Fatalf("Bar(-5) = %q, want clamped to 0%%", got)
	}
}

func TestRenderEmitsBarsOnChange(t *testing.T) {
	in := strings.NewReader("starting\n10%\n50%\n50%\nfinishing\n100%\n")
	var out strings.Builder
	progress.Render(in, &out, 10)
	s := out.String()
	for _, want := range []string{"10%", "50%", "100%"} {
		if !strings.Contains(s, want) {
			t.Errorf("Render output missing %q; got %q", want, s)
		}
	}
	// duplicate 50% should not draw twice
	if strings.Count(s, "50%") != 1 {
		t.Errorf("expected one 50%% bar, got %d in %q", strings.Count(s, "50%"), s)
	}
	if !strings.HasSuffix(s, "\n") {
		t.Errorf("Render output should end with newline, got %q", s)
	}
}
