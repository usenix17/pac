package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"starnix.net/pac/internal/cli"
	"starnix.net/pac/internal/run"
)

// runCLI captures stdout and stderr with an empty stdin.
func runCLI(args []string, r run.Runner) (code int, stdout, stderr string) {
	return runCLIStdin(args, r, "")
}

// runCLIStdin captures stdout and stderr, feeding stdin as the prompt input.
func runCLIStdin(args []string, r run.Runner, stdin string) (code int, stdout, stderr string) {
	var o, e bytes.Buffer
	code = cli.Run(args, r, strings.NewReader(stdin), &o, &e)
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

func TestInstallSubcommandDispatches(t *testing.T) {
	t.Setenv("PAC_PREFER", "system")
	t.Setenv("PAC_CONFIG", "")
	f := &run.Fake{Results: []run.Call{{}, {Out: ""}}} // pacman has it; flatpak no match
	code, _, _ := runCLI([]string{"install", "firefox"}, f)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	last := f.Calls[len(f.Calls)-1]
	if !reflect.DeepEqual(last, []string{"sudo", "pacman", "-S", "firefox"}) {
		t.Fatalf("last call = %v, want sudo pacman -S firefox", last)
	}
}

func TestSAliasMapsToInstall(t *testing.T) {
	t.Setenv("PAC_PREFER", "system")
	t.Setenv("PAC_CONFIG", "")
	f := &run.Fake{Results: []run.Call{{}, {Out: ""}}}
	code, _, _ := runCLI([]string{"-S", "firefox"}, f)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(f.Calls) == 0 || f.Calls[0][0] != "pacman" || f.Calls[0][1] != "-Si" {
		t.Fatalf("first call = %v, want pacman -Si probe", f.Calls)
	}
}

func TestInstallWithoutNameExits2(t *testing.T) {
	f := &run.Fake{}
	code, _, stderr := runCLI([]string{"install"}, f)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if len(f.Calls) != 0 {
		t.Fatalf("expected no backend calls, got %v", f.Calls)
	}
	if !strings.Contains(stderr, "package name") {
		t.Fatalf("stderr %q missing message", stderr)
	}
}

func TestInstallAskReadsStdin(t *testing.T) {
	t.Setenv("PAC_PREFER", "ask")
	t.Setenv("PAC_CONFIG", "")
	f := &run.Fake{Results: []run.Call{
		{}, // pacman has it
		{Out: "Discord\tChat\tcom.discordapp.Discord\t1.0\tstable\tflathub\n"},
	}}
	code, _, _ := runCLIStdin([]string{"install", "discord"}, f, "f\n")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	last := f.Calls[len(f.Calls)-1]
	if !reflect.DeepEqual(last, []string{"flatpak", "install", "com.discordapp.Discord"}) {
		t.Fatalf("last call = %v, want flatpak install (stdin chose flatpak)", last)
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
	if !strings.Contains(stdout, "[extra]") || !strings.Contains(stdout, "[flatpak]") {
		t.Fatalf("stdout missing source tags:\n%s", stdout)
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

func TestRemoveSubcommandDispatches(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{}}} // pacman -Qi: installed
	code, _, _ := runCLI([]string{"remove", "firefox"}, f)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	want := [][]string{{"pacman", "-Qi", "firefox"}, {"sudo", "pacman", "-R", "firefox"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestRAliasMapsToRemove(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{}}}
	code, _, _ := runCLI([]string{"-R", "firefox"}, f)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(f.Calls) == 0 || f.Calls[0][0] != "pacman" || f.Calls[0][1] != "-Qi" {
		t.Fatalf("first call = %v, want pacman -Qi probe", f.Calls)
	}
}

func TestRemoveWithoutNameExits2(t *testing.T) {
	f := &run.Fake{}
	code, _, stderr := runCLI([]string{"remove"}, f)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if len(f.Calls) != 0 {
		t.Fatalf("expected no backend calls, got %v", f.Calls)
	}
	if !strings.Contains(stderr, "package name") {
		t.Fatalf("stderr %q missing message", stderr)
	}
}

var errBoom = boomError("boom")

type boomError string

func (e boomError) Error() string { return string(e) }

func TestMirrorListDispatches(t *testing.T) {
	t.Setenv("PAC_CONFIG", "")
	dir := t.TempDir()
	t.Setenv("PAC_ALLOWLIST", filepath.Join(dir, "allowlist.yaml"))
	if err := os.WriteFile(filepath.Join(dir, "allowlist.yaml"),
		[]byte("data:\n  allowlist.yaml: |\n    packages:\n      - name: discord\n        approved: true\n        note: explicit\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	code, stdout, _ := runCLI([]string{"mirror", "list"}, &run.Fake{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "discord") {
		t.Fatalf("stdout missing discord: %q", stdout)
	}
}

func TestMirrorNoSubExits2(t *testing.T) {
	code, _, stderr := runCLI([]string{"mirror"}, &run.Fake{})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "subcommand") {
		t.Fatalf("stderr missing 'subcommand': %q", stderr)
	}
}
