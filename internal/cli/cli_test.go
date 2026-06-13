package cli_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"starnix.net/pac/internal/cli"
	"starnix.net/pac/internal/run"
)

func TestUpdateSubcommandDispatches(t *testing.T) {
	f := &run.Fake{}
	var out bytes.Buffer
	code := cli.Run([]string{"update"}, f, &out)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	want := [][]string{{"sudo", "pacman", "-Syu"}, {"flatpak", "update"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestSyuAliasMapsToUpdate(t *testing.T) {
	f := &run.Fake{}
	var out bytes.Buffer
	code := cli.Run([]string{"-Syu"}, f, &out)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	want := [][]string{{"sudo", "pacman", "-Syu"}, {"flatpak", "update"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestVersionNeedsNoRunner(t *testing.T) {
	var out bytes.Buffer
	code := cli.Run([]string{"--version"}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "pac "+cli.Version) {
		t.Fatalf("output %q missing version", out.String())
	}
}

func TestNoArgsShowsUsage(t *testing.T) {
	var out bytes.Buffer
	code := cli.Run(nil, nil, &out)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("output %q missing usage", out.String())
	}
}

func TestUnknownCommandExits2(t *testing.T) {
	var out bytes.Buffer
	code := cli.Run([]string{"frobnicate"}, nil, &out)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestUpdateFailureExits1(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Err: errBoom}}}
	var out bytes.Buffer
	if code := cli.Run([]string{"update"}, f, &out); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

var errBoom = boomError("boom")

type boomError string

func (e boomError) Error() string { return string(e) }
