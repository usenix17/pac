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

func TestBarStructure(t *testing.T) {
	b := progress.Bar(50, 10)
	if !strings.HasPrefix(b, "[") {
		t.Fatalf("Bar = %q, want leading [", b)
	}
	if !strings.Contains(b, "C") {
		t.Fatalf("Bar = %q, want a C pac-man", b)
	}
	if !strings.HasSuffix(b, "50%") {
		t.Fatalf("Bar = %q, want trailing 50%%", b)
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
