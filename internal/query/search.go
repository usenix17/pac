package query

import (
	"fmt"
	"strings"

	"starnix.net/pac/internal/run"
)

// Search queries pacman (official + aur-mirror) and flatpak for term and
// returns the merged hits (pacman first, then flatpak).
//
// Both Capture errors are intentionally ignored. The common case is a backend
// that finds nothing and exits non-zero; its empty output simply contributes
// no results. This also means a genuinely failed or missing backend binary
// (e.g. flatpak not installed) is silently treated as "no hits from that
// source" rather than an error. That is acceptable for now; surfacing genuine
// backend failures as warnings is deferred to when the CLI gains a stderr
// writer (Phase 2 slice 2).
func Search(r run.Runner, term string) []Result {
	pac, _ := r.Capture("pacman", "-Ss", term)
	flat, _ := r.Capture("flatpak", "search", term)
	return append(ParsePacman(pac), ParseFlatpak(flat)...)
}

// Format renders results as source-tagged lines.
func Format(results []Result) string {
	if len(results) == 0 {
		return "no results\n"
	}
	var b strings.Builder
	for _, r := range results {
		installed := ""
		if r.Installed {
			installed = " [installed]"
		}
		fmt.Fprintf(&b, "[%s] %s %s%s\n", r.Source, r.Name, r.Version, installed)
		if r.Desc != "" {
			fmt.Fprintf(&b, "    %s\n", r.Desc)
		}
	}
	return b.String()
}
