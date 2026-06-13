package query_test

import (
	"reflect"
	"testing"

	"starnix.net/pac/internal/query"
)

func TestParsePacmanTwoResults(t *testing.T) {
	out := "extra/firefox 151.0.4-1 [installed]\n" +
		"    Fast, Private & Safe Web Browser\n" +
		"aur-mirror/claude-code 2.1.176-1\n" +
		"    An agentic coding tool that lives in your terminal\n"

	got := query.ParsePacman(out)
	want := []query.Result{
		{Source: "extra", Name: "firefox", Version: "151.0.4-1", Desc: "Fast, Private & Safe Web Browser", Installed: true},
		{Source: "aur-mirror", Name: "claude-code", Version: "2.1.176-1", Desc: "An agentic coding tool that lives in your terminal", Installed: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParsePacman =\n%#v\nwant\n%#v", got, want)
	}
}

func TestParsePacmanEmpty(t *testing.T) {
	if got := query.ParsePacman(""); len(got) != 0 {
		t.Fatalf("ParsePacman(\"\") = %v, want empty", got)
	}
}

func TestParsePacmanHeaderWithoutDescription(t *testing.T) {
	// A trailing header with no following indented line must still parse.
	got := query.ParsePacman("core/bash 5.2-1\n")
	want := []query.Result{{Source: "core", Name: "bash", Version: "5.2-1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
