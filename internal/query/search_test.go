package query_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"starnix.net/pac/internal/query"
	"starnix.net/pac/internal/run"
)

func TestSearchQueriesBothBackends(t *testing.T) {
	f := &run.Fake{Results: []run.Call{
		{Out: "extra/firefox 151.0.4-1 [installed]\n    Fast, Private & Safe Web Browser\n"},
		{Out: "Firefox\tWeb browser\torg.mozilla.firefox\t151.0.4\tstable\tflathub\n"},
	}}

	got := query.Search(f, "firefox")

	wantCalls := [][]string{
		{"pacman", "-Ss", "firefox"},
		{"flatpak", "search", "firefox"},
	}
	if !reflect.DeepEqual(f.Calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", f.Calls, wantCalls)
	}
	want := []query.Result{
		{Source: "extra", Name: "firefox", Version: "151.0.4-1", Desc: "Fast, Private & Safe Web Browser", Installed: true},
		{Source: "flatpak", Name: "org.mozilla.firefox", Version: "151.0.4", Desc: "Web browser"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Search =\n%#v\nwant\n%#v", got, want)
	}
}

func TestFormatTagsBySource(t *testing.T) {
	results := []query.Result{
		{Source: "aur-mirror", Name: "claude-code", Version: "2.1.176-1", Desc: "agentic tool", Installed: true},
		{Source: "flatpak", Name: "org.signal.Signal", Version: "7.0", Desc: "private messenger"},
	}
	out := query.Format(results)
	for _, want := range []string{"[aur-mirror]", "claude-code", "2.1.176-1", "[installed]", "[flatpak]", "org.signal.Signal", "private messenger"} {
		if !strings.Contains(out, want) {
			t.Errorf("Format output missing %q; got:\n%s", want, out)
		}
	}
}

func TestFormatEmpty(t *testing.T) {
	if out := query.Format(nil); !strings.Contains(out, "no results") {
		t.Fatalf("Format(nil) = %q, want a 'no results' message", out)
	}
}

func TestSearchIgnoresBackendErrors(t *testing.T) {
	// A backend that errors (no-match exit, or even a missing binary) must not
	// fail the search; its source simply contributes no results.
	f := &run.Fake{Results: []run.Call{
		{Out: "", Err: errors.New("exit status 1")},
		{Out: "Sig\tmsg\torg.signal.Signal\t7.0\tstable\tflathub\n"},
	}}
	got := query.Search(f, "signal")
	want := []query.Result{{Source: "flatpak", Name: "org.signal.Signal", Version: "7.0", Desc: "msg"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Search =\n%#v\nwant\n%#v", got, want)
	}
}
