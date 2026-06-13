package query_test

import (
	"reflect"
	"testing"

	"starnix.net/pac/internal/query"
)

func TestParseFlatpakResults(t *testing.T) {
	// Tab-separated: Name, Description, AppID, Version, Branch, Remotes
	out := "Firefox\tFast, Private & Safe Web Browser\torg.mozilla.firefox\t151.0.4\tstable\tflathub\n" +
		"Floorp\tCustomizable Firefox-based browser\tone.ablaze.floorp\t12.14.2\tstable\tflathub\n"

	got := query.ParseFlatpak(out)
	want := []query.Result{
		{Source: "flatpak", Name: "org.mozilla.firefox", Version: "151.0.4", Desc: "Fast, Private & Safe Web Browser"},
		{Source: "flatpak", Name: "one.ablaze.floorp", Version: "12.14.2", Desc: "Customizable Firefox-based browser"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseFlatpak =\n%#v\nwant\n%#v", got, want)
	}
}

func TestParseFlatpakSkipsNonTabLines(t *testing.T) {
	// "No matches found" and blank lines have no tabs and must be skipped.
	if got := query.ParseFlatpak("No matches found\n\n"); len(got) != 0 {
		t.Fatalf("ParseFlatpak = %v, want empty", got)
	}
}
