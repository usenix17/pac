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

const flatpakRow = "Discord\tChat\tcom.discordapp.Discord\t1.0\tstable\tflathub\n"

func lastCall(f *run.Fake) []string { return f.Calls[len(f.Calls)-1] }

func TestInstallPacmanOnly(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{}, {Out: ""}}} // pacman has it; flatpak no match
	code := cmd.Install(f, "system", "firefox", strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	want := [][]string{
		{"pacman", "-Si", "firefox"},
		{"flatpak", "search", "firefox"},
		{"sudo", "pacman", "-S", "firefox"},
	}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("Calls = %v, want %v", f.Calls, want)
	}
}

func TestInstallFlatpakOnlyByAppID(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Err: errors.New("not in repos")}, {Out: flatpakRow}}}
	code := cmd.Install(f, "system", "com.discordapp.Discord", strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := lastCall(f); !reflect.DeepEqual(got, []string{"flatpak", "install", "--noninteractive", "com.discordapp.Discord"}) {
		t.Fatalf("last call = %v, want flatpak install com.discordapp.Discord", got)
	}
}

func TestInstallFlatpakMatchesHumanName(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Err: errors.New("not in repos")}, {Out: flatpakRow}}}
	// "discord" matches the human name "Discord" case-insensitively.
	code := cmd.Install(f, "system", "discord", strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := lastCall(f); !reflect.DeepEqual(got, []string{"flatpak", "install", "--noninteractive", "com.discordapp.Discord"}) {
		t.Fatalf("last call = %v, want flatpak install com.discordapp.Discord", got)
	}
}

func TestInstallBothPreferSystem(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{}, {Out: flatpakRow}}}
	code := cmd.Install(f, "system", "discord", strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := lastCall(f); !reflect.DeepEqual(got, []string{"sudo", "pacman", "-S", "discord"}) {
		t.Fatalf("last call = %v, want sudo pacman -S discord", got)
	}
}

func TestInstallBothPreferFlatpak(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{}, {Out: flatpakRow}}}
	code := cmd.Install(f, "flatpak", "discord", strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := lastCall(f); !reflect.DeepEqual(got, []string{"flatpak", "install", "--noninteractive", "com.discordapp.Discord"}) {
		t.Fatalf("last call = %v, want flatpak install com.discordapp.Discord", got)
	}
}

func TestInstallBothAskChoosesFlatpak(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{}, {Out: flatpakRow}}}
	var out bytes.Buffer
	code := cmd.Install(f, "ask", "discord", strings.NewReader("f\n"), &out, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := lastCall(f); !reflect.DeepEqual(got, []string{"flatpak", "install", "--noninteractive", "com.discordapp.Discord"}) {
		t.Fatalf("last call = %v, want flatpak install", got)
	}
	if !strings.Contains(out.String(), "[s/f]") {
		t.Fatalf("prompt not shown to stdout: %q", out.String())
	}
}

func TestInstallBothAskDefaultsToSystem(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{}, {Out: flatpakRow}}}
	// "s" (and anything that is not f) resolves to system.
	code := cmd.Install(f, "ask", "discord", strings.NewReader("s\n"), &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := lastCall(f); !reflect.DeepEqual(got, []string{"sudo", "pacman", "-S", "discord"}) {
		t.Fatalf("last call = %v, want sudo pacman -S discord", got)
	}
}

func TestInstallNotFoundSuggestsMirrorAdd(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Err: errors.New("not in repos")}, {Out: "No matches found\n"}}}
	var errb bytes.Buffer
	code := cmd.Install(f, "system", "obscurepkg", strings.NewReader(""), &bytes.Buffer{}, &errb)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if len(f.Calls) != 2 { // only the two probes; nothing installed
		t.Fatalf("expected only 2 probe calls, got %v", f.Calls)
	}
	if !strings.Contains(errb.String(), "pac mirror add obscurepkg") {
		t.Fatalf("stderr missing mirror-add suggestion: %q", errb.String())
	}
}

func TestInstallFailureReturns1(t *testing.T) {
	f := &run.Fake{Results: []run.Call{
		{},                        // pacman -Si: found
		{Out: ""},                 // flatpak: none
		{Err: errors.New("boom")}, // sudo pacman -S fails
	}}
	var errb bytes.Buffer
	code := cmd.Install(f, "system", "firefox", strings.NewReader(""), &bytes.Buffer{}, &errb)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "install failed") {
		t.Fatalf("stderr missing failure message: %q", errb.String())
	}
}
