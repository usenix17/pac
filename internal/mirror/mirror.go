// Package mirror reads and edits the aur-mirror allowlist and resolves AUR
// dependency closures (via `aur depends` through a run.Runner). The allowlist
// is edited line-based to avoid a YAML dependency, matching its regular
// "- name: X / approved: true / note: ..." entry structure.
package mirror

import (
	"bufio"
	"errors"
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
