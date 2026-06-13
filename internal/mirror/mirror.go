// Package mirror reads and edits the aur-mirror allowlist and resolves AUR
// dependency closures (via `aur depends` through a run.Runner). The allowlist
// is edited line-based to avoid a YAML dependency, matching its regular
// "- name: X / approved: true / note: ..." entry structure.
package mirror

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"starnix.net/pac/internal/run"
)

// ApprovedNames returns the package names in the allowlist (each "- name: X"
// entry), sorted. The ConfigMap's own "name:" line (no "- " prefix) is ignored.
func ApprovedNames(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var names []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "- name:") {
			names = append(names, strings.TrimSpace(strings.TrimPrefix(line, "- name:")))
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// Closure resolves the AUR dependency closure of pkgs using resolver (e.g.
// {"aur","depends","-n"} for local aurutils, or a docker invocation). It runs
// resolver+pkgs through the runner and returns the unique sorted package names.
// `aur depends -n` emits tab-separated "target<TAB>dep" lines.
func Closure(r run.Runner, resolver, pkgs []string) ([]string, error) {
	if len(resolver) == 0 {
		return nil, errors.New("mirror: resolver must have at least one element")
	}
	args := append(append([]string{}, resolver[1:]...), pkgs...)
	out, err := r.Capture(resolver[0], args...)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		for _, field := range strings.Split(line, "\t") {
			if name := strings.TrimSpace(field); name != "" {
				set[name] = true
			}
		}
	}
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// Missing returns the closure members not already in approved, preserving
// closure order.
func Missing(closure, approved []string) []string {
	have := make(map[string]bool, len(approved))
	for _, a := range approved {
		have[a] = true
	}
	var missing []string
	for _, c := range closure {
		if !have[c] {
			missing = append(missing, c)
		}
	}
	return missing
}

// AppendEntries appends allowlist entries (6-space indented, matching the
// generated format) for each name. requested marks which names are explicitly
// requested (note: explicit) versus pulled-in dependencies (note: dependency).
// Entries are appended at end of file; the caller reviews the git diff.
func AppendEntries(path string, names []string, requested map[string]bool) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, name := range names {
		note := "dependency"
		if requested[name] {
			note = "explicit"
		}
		if _, err := fmt.Fprintf(f, "      - name: %s\n        approved: true\n        note: %s\n", name, note); err != nil {
			return err
		}
	}
	return nil
}

// ExplicitNames returns the allowlist package names marked "note: explicit",
// sorted. These are the "roots" (directly-wanted packages).
func ExplicitNames(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var names []string
	var current string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "- name:"):
			current = strings.TrimSpace(strings.TrimPrefix(line, "- name:"))
		case line == "note: explicit" && current != "":
			names = append(names, current)
			current = ""
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// RemoveEntries rewrites the allowlist without the 3-line blocks (the
// "- name: X" line plus the following two: approved, note) for the given names.
func RemoveEntries(path string, names []string) error {
	remove := make(map[string]bool, len(names))
	for _, n := range names {
		remove[n] = true
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "- name:") {
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:"))
			if remove[name] {
				i += 2 // also skip the approved and note lines (3-line block)
				continue
			}
		}
		out = append(out, lines[i])
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}
