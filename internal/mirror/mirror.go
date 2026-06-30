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
	"path/filepath"
	"sort"
	"strings"

	"starnix.net/pac/internal/run"
	"starnix.net/pac/internal/validate"
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
	// "--" ends the resolver's option parsing so a pkg beginning with '-' is
	// treated as a target, not a flag.
	args := append(append(append([]string{}, resolver[1:]...), "--"), pkgs...)
	out, err := r.Capture(resolver[0], args...)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		for _, field := range strings.Split(line, "\t") {
			name := strings.TrimSpace(field)
			if name == "" {
				continue
			}
			// The resolver's stdout becomes allowlist content (a GitOps trust
			// root), so every token is charset-validated before we trust it.
			if err := validate.PkgName(name); err != nil {
				return nil, fmt.Errorf("resolver returned %v", err)
			}
			set[name] = true
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

// atomicWrite replaces path's contents with data atomically: it writes a temp
// file in the SAME directory, flushes it to stable storage, then renames it
// over path. The original file's permission bits are preserved (a plain
// os.WriteFile would reset them to 0644). The allowlist is a GitOps trust root,
// so it must never be observed half-written.
func atomicWrite(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".allowlist-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup of the temp file if we bail before the rename.
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	tmpName = "" // renamed into place; nothing to clean up
	return nil
}

// AppendEntries appends allowlist entries (6-space indented, matching the
// generated format) for each name. requested marks which names are explicitly
// requested (note: explicit) versus pulled-in dependencies (note: dependency).
// Entries are appended at end of file; the caller reviews the git diff. The
// rewrite is atomic (see atomicWrite).
func AppendEntries(path string, names []string, requested map[string]bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.Write(data)
	for _, name := range names {
		note := "dependency"
		if requested[name] {
			note = "explicit"
		}
		fmt.Fprintf(&b, "      - name: %s\n        approved: true\n        note: %s\n", name, note)
	}
	return atomicWrite(path, []byte(b.String()))
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
	return atomicWrite(path, []byte(strings.Join(out, "\n")))
}

// Orphaned determines which of pkgs (and their AUR deps) can be removed from
// the allowlist: the members of pkgs' closure that are NOT in the closure of
// the remaining explicit roots, intersected with what is actually approved.
// kept lists requested pkgs that are still required by a remaining root (so
// they are protected, not removed).
func Orphaned(r run.Runner, resolver []string, allowlistPath string, pkgs []string) (removable, kept []string, err error) {
	target, err := Closure(r, resolver, pkgs)
	if err != nil {
		return nil, nil, err
	}
	explicit, err := ExplicitNames(allowlistPath)
	if err != nil {
		return nil, nil, err
	}
	removing := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		removing[p] = true
	}
	var roots []string
	for _, e := range explicit {
		if !removing[e] {
			roots = append(roots, e)
		}
	}
	var needed []string
	if len(roots) > 0 {
		if needed, err = Closure(r, resolver, roots); err != nil {
			return nil, nil, err
		}
	}
	neededSet := make(map[string]bool, len(needed))
	for _, n := range needed {
		neededSet[n] = true
	}
	approved, err := ApprovedNames(allowlistPath)
	if err != nil {
		return nil, nil, err
	}
	approvedSet := make(map[string]bool, len(approved))
	for _, a := range approved {
		approvedSet[a] = true
	}
	for _, name := range target {
		if !neededSet[name] && approvedSet[name] {
			removable = append(removable, name)
		}
	}
	removableSet := make(map[string]bool, len(removable))
	for _, n := range removable {
		removableSet[n] = true
	}
	for _, p := range pkgs {
		if !removableSet[p] && neededSet[p] {
			kept = append(kept, p)
		}
	}
	return removable, kept, nil
}
