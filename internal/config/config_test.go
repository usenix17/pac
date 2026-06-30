package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"starnix.net/pac/internal/config"
)

// isolate clears the env vars Load reads so tests don't leak into each other.
func isolate(t *testing.T, home string) {
	t.Setenv("HOME", home)
	t.Setenv("PAC_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "empty"))
	t.Setenv("PAC_ALLOWLIST", "")
	t.Setenv("PAC_BUILDER_IMAGE", "")
	t.Setenv("PAC_PREFER", "")
}

func TestLoadDefaults(t *testing.T) {
	isolate(t, "/home/x")
	c := config.Load()
	if c.Allowlist != "/home/x/ArgoCD/applications/aur-mirror/allowlist.yaml" {
		t.Errorf("Allowlist = %q", c.Allowlist)
	}
	// The default builder image must be digest-pinned (never a floating tag),
	// so push access to the registry can't swap the signer's code. The exact
	// digest rotates via pin-image.sh, so assert the shape, not a literal.
	if !strings.HasPrefix(c.BuilderImage, "registry.starnix.net/library/aur-builder@sha256:") {
		t.Errorf("BuilderImage = %q, want a digest-pinned aur-builder ref", c.BuilderImage)
	}
	if c.Prefer != "system" {
		t.Errorf("Prefer = %q, want system", c.Prefer)
	}
}

func TestLoadFromFileWithTilde(t *testing.T) {
	isolate(t, "/home/y")
	cfg := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(cfg, []byte("# comment\nprefer = flatpak\nallowlist = ~/my.yaml\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("PAC_CONFIG", cfg)

	c := config.Load()
	if c.Prefer != "flatpak" {
		t.Errorf("Prefer = %q, want flatpak", c.Prefer)
	}
	if c.Allowlist != "/home/y/my.yaml" {
		t.Errorf("Allowlist = %q, want /home/y/my.yaml (tilde expanded)", c.Allowlist)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	isolate(t, "/home/z")
	cfg := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(cfg, []byte("prefer = flatpak\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("PAC_CONFIG", cfg)
	t.Setenv("PAC_PREFER", "ask")

	if c := config.Load(); c.Prefer != "ask" {
		t.Errorf("Prefer = %q, want ask (env wins)", c.Prefer)
	}
}
