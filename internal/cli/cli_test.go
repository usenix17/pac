package cli_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"starnix.net/pac/internal/cli"
	"starnix.net/pac/internal/run"
)

// runCLI is a helper that captures stdout and stderr separately.
func runCLI(args []string, r run.Runner) (code int, stdout, stderr string) {
	var o, e bytes.Buffer
	code = cli.Run(args, r, &o, &e)
	return code, o.String(), e.String()
}

func TestUpdateSubcommandDispatches(t *testing.T) {
	f := &run.Fake{}
	code, _, _ := runCLI([]string{"update"}, f)
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
	code, _, _ := runCLI([]string{"-Syu"}, f)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	want := [][]string{{"sudo", "pacman", "-Syu"}, {"flatpak", "update"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestVersionToStdout(t *testing.T) {
	code, stdout, _ := runCLI([]string{"--version"}, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "pac "+cli.Version) {
		t.Fatalf("stdout %q missing version", stdout)
	}
}

func TestNoArgsShowsUsageOnStdout(t *testing.T) {
	code, stdout, _ := runCLI(nil, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("stdout %q missing usage", stdout)
	}
}

func TestHelpFlagShowsUsageOnStdout(t *testing.T) {
	code, stdout, _ := runCLI([]string{"--help"}, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("stdout %q missing usage", stdout)
	}
}

func TestUnknownCommandErrorToStderr(t *testing.T) {
	code, stdout, stderr := runCLI([]string{"frobnicate"}, nil)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("stderr %q missing 'unknown command'", stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout should be empty on error, got %q", stdout)
	}
}

func TestUpdateFailureErrorToStderr(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Err: errBoom}}}
	code, _, stderr := runCLI([]string{"update"}, f)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "update failed") {
		t.Fatalf("stderr %q missing 'update failed'", stderr)
	}
}

func TestSearchSubcommandPrintsToStdout(t *testing.T) {
	f := &run.Fake{Results: []run.Call{
		{Out: "extra/firefox 151.0.4-1 [installed]\n    Fast, Private & Safe Web Browser\n"},
		{Out: "Firefox\tWeb browser\torg.mozilla.firefox\t151.0.4\tstable\tflathub\n"},
	}}
	code, stdout, _ := runCLI([]string{"search", "firefox"}, f)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	wantCalls := [][]string{{"pacman", "-Ss", "firefox"}, {"flatpak", "search", "firefox"}}
	if !reflect.DeepEqual(f.Calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", f.Calls, wantCalls)
	}
	if !strings.Contains(stdout, "[extra]") {
		t.Fatalf("stdout missing [extra]:\n%s", stdout)
	}
	if !strings.Contains(stdout, "[flatpak]") {
		t.Fatalf("stdout missing [flatpak]:\n%s", stdout)
	}
}

func TestSsAliasMapsToSearch(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Out: ""}, {Out: ""}}}
	code, _, _ := runCLI([]string{"-Ss", "firefox"}, f)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	wantCalls := [][]string{{"pacman", "-Ss", "firefox"}, {"flatpak", "search", "firefox"}}
	if !reflect.DeepEqual(f.Calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", f.Calls, wantCalls)
	}
}

func TestSearchJoinsMultiWordTerm(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Out: ""}, {Out: ""}}}
	code, _, _ := runCLI([]string{"search", "web", "browser"}, f)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	wantCalls := [][]string{{"pacman", "-Ss", "web browser"}, {"flatpak", "search", "web browser"}}
	if !reflect.DeepEqual(f.Calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", f.Calls, wantCalls)
	}
}

func TestSearchWithoutTermErrorToStderr(t *testing.T) {
	f := &run.Fake{}
	code, _, stderr := runCLI([]string{"search"}, f)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if len(f.Calls) != 0 {
		t.Fatalf("expected no backend calls, got %v", f.Calls)
	}
	if !strings.Contains(stderr, "term") {
		t.Fatalf("stderr %q missing term-required message", stderr)
	}
}

func TestSearchEmptyTermErrorToStderr(t *testing.T) {
	f := &run.Fake{}
	code, _, stderr := runCLI([]string{"search", "   "}, f)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if len(f.Calls) != 0 {
		t.Fatalf("expected no backend calls for blank term, got %v", f.Calls)
	}
	if !strings.Contains(stderr, "term") {
		t.Fatalf("stderr %q missing term-required message", stderr)
	}
}

var errBoom = boomError("boom")

type boomError string

func (e boomError) Error() string { return string(e) }
