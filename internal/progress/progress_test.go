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
	progress.Render(in, &out, 10, "")
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

// A realistic flatpak stream: a numbered ref table, then two items each going
// 0->100%. Each item's ref should label its own bar line.
func TestRenderLabelsItemsFromStream(t *testing.T) {
	in := strings.NewReader(
		" 1.\t   \torg.gnome.Calculator.Locale\tstable\ti\tflathub\t< 1.6 MB\n" +
			" 2.\t   \torg.gnome.Calculator\tstable\ti\tflathub\t< 1.8 MB\n" +
			"Installing 1/2…\n" +
			"Installing 1/2…                        0%  0 bytes/s\n" +
			"Installing 1/2… ███ 100%\n" +
			"Installing 2/2…\n" +
			"Installing 2/2…                        0%  0 bytes/s\n" +
			"Installing 2/2… ███ 100%\n" +
			"Installation complete.\n")
	var out strings.Builder
	progress.Render(in, &out, 10, "")
	s := out.String()
	for _, want := range []string{"org.gnome.Calculator.Locale", "org.gnome.Calculator"} {
		if !strings.Contains(s, want) {
			t.Errorf("Render output missing ref %q; got %q", want, s)
		}
	}
	// One \n per finished item -> two lines.
	if n := strings.Count(s, "\n"); n != 2 {
		t.Errorf("expected 2 item lines, got %d newlines in %q", n, s)
	}
}

// When the caller supplies a label and the stream has no ref table (single
// install), that label is shown beside the bar.
func TestRenderUsesCallerLabel(t *testing.T) {
	in := strings.NewReader("Installing 1/1…\nInstalling 1/1… 50%\n")
	var out strings.Builder
	progress.Render(in, &out, 10, "com.vivaldi.Vivaldi")
	if s := out.String(); !strings.Contains(s, "com.vivaldi.Vivaldi") {
		t.Errorf("Render output missing caller label; got %q", s)
	}
}

// No table and no caller label: fall back to flatpak's N/M counter.
func TestRenderFallsBackToCounter(t *testing.T) {
	in := strings.NewReader("Installing 2/3…\nInstalling 2/3… 50%\n")
	var out strings.Builder
	progress.Render(in, &out, 10, "")
	if s := out.String(); !strings.Contains(s, "2/3") {
		t.Errorf("Render output missing N/M counter; got %q", s)
	}
}
