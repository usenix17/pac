package preflight_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"starnix.net/pac/internal/preflight"
)

func check(t *testing.T, conf string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "pacman.conf")
	if err := os.WriteFile(p, []byte(conf), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	var b bytes.Buffer
	preflight.SigLevelCheck(p, &b)
	return b.String()
}

func TestSigLevelRequiredNoWarning(t *testing.T) {
	conf := "[options]\nSigLevel = Required DatabaseOptional\n\n[aur-mirror]\nSigLevel = Required\nServer = file:///srv/aur\n"
	if got := check(t, conf); got != "" {
		t.Fatalf("expected no warning, got %q", got)
	}
}

func TestSigLevelOptionalWarns(t *testing.T) {
	conf := "[aur-mirror]\nSigLevel = Optional TrustAll\nServer = file:///srv/aur\n"
	if got := check(t, conf); !strings.Contains(got, "aur-mirror") || !strings.Contains(got, "WARNING") {
		t.Fatalf("expected aur-mirror SigLevel warning, got %q", got)
	}
}

func TestSigLevelTrustAllWarnsEvenWithRequired(t *testing.T) {
	conf := "[aur-mirror]\nSigLevel = Required TrustAll\n"
	if got := check(t, conf); !strings.Contains(got, "WARNING") {
		t.Fatalf("TrustAll should warn despite Required, got %q", got)
	}
}

func TestSigLevelInheritsGlobalRequired(t *testing.T) {
	// No SigLevel under [aur-mirror]: it inherits [options]'s Required.
	conf := "[options]\nSigLevel = Required\n\n[aur-mirror]\nServer = file:///srv/aur\n"
	if got := check(t, conf); got != "" {
		t.Fatalf("expected inherited Required, no warning, got %q", got)
	}
}

func TestSigLevelUnsetWarns(t *testing.T) {
	conf := "[aur-mirror]\nServer = file:///srv/aur\n"
	if got := check(t, conf); !strings.Contains(got, "unset") {
		t.Fatalf("expected unset warning, got %q", got)
	}
}

func TestNoAurMirrorSectionNoWarning(t *testing.T) {
	conf := "[options]\nSigLevel = Optional\n\n[core]\nInclude = /etc/pacman.d/mirrorlist\n"
	if got := check(t, conf); got != "" {
		t.Fatalf("absent aur-mirror should not warn, got %q", got)
	}
}

func TestPublicAURRepoWarns(t *testing.T) {
	conf := "[chaotic-aur]\nInclude = /etc/pacman.d/chaotic-mirrorlist\n"
	if got := check(t, conf); !strings.Contains(got, "chaotic-aur") {
		t.Fatalf("expected public-AUR repo warning, got %q", got)
	}
}

func TestMissingFileSilent(t *testing.T) {
	var b bytes.Buffer
	preflight.SigLevelCheck(filepath.Join(t.TempDir(), "nope.conf"), &b)
	if b.String() != "" {
		t.Fatalf("missing file should be silent, got %q", b.String())
	}
}
