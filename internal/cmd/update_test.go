package cmd_test

import (
	"errors"
	"reflect"
	"testing"

	"starnix.net/pac/internal/cmd"
	"starnix.net/pac/internal/run"
)

func TestUpdateRunsPacmanThenFlatpak(t *testing.T) {
	f := &run.Fake{}
	if err := cmd.Update(f); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := [][]string{
		{"sudo", "pacman", "-Syu"},
		{"flatpak", "update"},
	}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestUpdateStopsIfPacmanFails(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Err: errors.New("pacman exploded")}}}
	if err := cmd.Update(f); err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(f.Calls) != 1 {
		t.Fatalf("expected only pacman to run, got %v", f.Calls)
	}
}
