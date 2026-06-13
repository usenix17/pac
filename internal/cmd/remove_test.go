package cmd_test

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"starnix.net/pac/internal/cmd"
	"starnix.net/pac/internal/run"
)

func TestRemovePacmanInstalled(t *testing.T) {
	f := &run.Fake{Results: []run.Call{
		{}, // pacman -Qi: installed
		{}, // sudo pacman -R: succeeds
	}}
	code := cmd.Remove(f, "firefox", &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	want := [][]string{
		{"pacman", "-Qi", "firefox"},
		{"sudo", "pacman", "-R", "firefox"},
	}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestRemoveFlatpakInstalled(t *testing.T) {
	f := &run.Fake{Results: []run.Call{
		{Err: errors.New("not installed via pacman")}, // pacman -Qi fails
		{},                                            // flatpak info: installed
		{},                                            // flatpak uninstall: succeeds
	}}
	code := cmd.Remove(f, "com.discordapp.Discord", &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	want := [][]string{
		{"pacman", "-Qi", "com.discordapp.Discord"},
		{"flatpak", "info", "com.discordapp.Discord"},
		{"flatpak", "uninstall", "--noninteractive", "com.discordapp.Discord"},
	}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestRemoveNotInstalled(t *testing.T) {
	f := &run.Fake{Results: []run.Call{
		{Err: errors.New("no")}, // pacman -Qi fails
		{Err: errors.New("no")}, // flatpak info fails
	}}
	var errb bytes.Buffer
	code := cmd.Remove(f, "ghostpkg", &errb)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if len(f.Calls) != 2 { // only the two probes; nothing removed
		t.Fatalf("expected only 2 probe calls, got %v", f.Calls)
	}
	if !strings.Contains(errb.String(), "not installed") {
		t.Fatalf("stderr missing 'not installed': %q", errb.String())
	}
}

func TestRemoveFailureReturns1(t *testing.T) {
	f := &run.Fake{Results: []run.Call{
		{},                        // pacman -Qi: installed
		{Err: errors.New("boom")}, // sudo pacman -R fails
	}}
	var errb bytes.Buffer
	code := cmd.Remove(f, "firefox", &errb)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "remove failed") {
		t.Fatalf("stderr missing 'remove failed': %q", errb.String())
	}
}
