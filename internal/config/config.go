// Package config resolves pac settings as env var > config file > default.
package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Config holds resolved pac settings.
type Config struct {
	Allowlist    string // path to the aur-mirror allowlist.yaml
	BuilderImage string // image providing `aur` for closure resolution
	Prefer       string // disambiguation default: system | flatpak | ask
}

func configPath() string {
	if p := os.Getenv("PAC_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(base, "pac", "config")
}

// parseFile reads an INI-ish "key = value" file ('#' comments, blank lines).
func parseFile(path string) map[string]string {
	m := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return m
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	// Best-effort: a scan error (e.g. an overlong or truncated line) leaves the
	// keys parsed so far; missing keys fall back to defaults in resolve.
	_ = sc.Err()
	return m
}

func expandTilde(p string) string {
	home := os.Getenv("HOME")
	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, p[2:])
	default:
		return p
	}
}

func resolve(env string, file map[string]string, key, def string) string {
	if v := os.Getenv(env); v != "" {
		return expandTilde(v)
	}
	if v, ok := file[key]; ok && v != "" {
		return expandTilde(v)
	}
	return def
}

// Load resolves settings. A missing config file is not an error (defaults win).
func Load() Config {
	file := parseFile(configPath())
	home := os.Getenv("HOME")
	return Config{
		Allowlist:    resolve("PAC_ALLOWLIST", file, "allowlist", filepath.Join(home, "ArgoCD/applications/aur-mirror/allowlist.yaml")),
		BuilderImage: resolve("PAC_BUILDER_IMAGE", file, "builder_image", "registry.starnix.net/library/aur-builder:latest"),
		Prefer:       resolve("PAC_PREFER", file, "prefer", "system"),
	}
}
