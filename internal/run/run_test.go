package run_test

import (
	"errors"
	"reflect"
	"testing"

	"starnix.net/pac/internal/run"
)

func TestFakeRecordsCalls(t *testing.T) {
	f := &run.Fake{}
	_ = f.Run("pacman", "-Syu")
	_, _ = f.Capture("flatpak", "search", "signal")

	want := [][]string{
		{"pacman", "-Syu"},
		{"flatpak", "search", "signal"},
	}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestFakeReturnsQueuedResults(t *testing.T) {
	boom := errors.New("boom")
	f := &run.Fake{Results: []run.Call{{Err: boom}, {Out: "hello"}}}

	if err := f.Run("a"); !errors.Is(err, boom) {
		t.Fatalf("Run err = %v, want %v", err, boom)
	}
	out, err := f.Capture("b")
	if err != nil || out != "hello" {
		t.Fatalf("Capture = (%q, %v), want (\"hello\", nil)", out, err)
	}
}

func TestRealCaptureRunsCommand(t *testing.T) {
	out, err := run.Real{}.Capture("echo", "candy")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out != "candy\n" {
		t.Fatalf("out = %q, want %q", out, "candy\n")
	}
}

func TestFakeRunBarRecordsCall(t *testing.T) {
	f := &run.Fake{}
	// First arg is the display label; it is not part of the recorded command.
	if err := f.RunBar("com.x.Y", "flatpak", "install", "-y", "com.x.Y"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := [][]string{{"flatpak", "install", "-y", "com.x.Y"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}
