package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"starnix.net/pac/internal/config"
	"starnix.net/pac/internal/mirror"
	"starnix.net/pac/internal/run"
	"starnix.net/pac/internal/validate"
)

// validatePkgs rejects any user-supplied package name outside the Arch pkgname
// charset before it reaches the resolver (aur/docker). Returns the first error.
func validatePkgs(pkgs []string) error {
	for _, p := range pkgs {
		if err := validate.PkgName(p); err != nil {
			return err
		}
	}
	return nil
}

// resolverPrefix returns the `aur depends -n` invocation: local aurutils if
// present, otherwise via the builder image in docker.
func resolverPrefix(image string) []string {
	if _, err := exec.LookPath("aur"); err == nil {
		return []string{"aur", "depends", "-n"}
	}
	return []string{"docker", "run", "--rm", image, "aur", "depends", "-n"}
}

// Mirror handles `pac mirror <sub> [pkg...]`: list, add, show, remove. It edits
// the allowlist file (cfg.Allowlist); the user reviews the git diff and pushes.
func Mirror(r run.Runner, cfg config.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "pac: mirror requires a subcommand (add|remove|show|list)")
		return 2
	}
	sub, pkgs := args[0], args[1:]

	switch sub {
	case "list":
		names, err := mirror.ApprovedNames(cfg.Allowlist)
		if err != nil {
			fmt.Fprintf(stderr, "pac: %v\n", err)
			return 1
		}
		for _, n := range names {
			fmt.Fprintln(stdout, n)
		}
		return 0

	case "add", "show":
		if len(pkgs) == 0 {
			fmt.Fprintf(stderr, "pac: mirror %s requires a package name\n", sub)
			return 2
		}
		if err := validatePkgs(pkgs); err != nil {
			fmt.Fprintf(stderr, "pac: %v\n", err)
			return 2
		}
		closure, err := mirror.Closure(r, resolverPrefix(cfg.BuilderImage), pkgs)
		if err != nil {
			fmt.Fprintf(stderr, "pac: aur closure resolution failed: %v\n", err)
			return 1
		}
		approved, err := mirror.ApprovedNames(cfg.Allowlist)
		if err != nil {
			fmt.Fprintf(stderr, "pac: %v\n", err)
			return 1
		}
		missing := mirror.Missing(closure, approved)
		if len(missing) == 0 {
			fmt.Fprintf(stdout, "%s and all AUR deps are already in the allowlist\n", strings.Join(pkgs, ", "))
			return 0
		}
		if sub == "show" {
			fmt.Fprintf(stdout, "%d package(s) would be added:\n", len(missing))
			for _, n := range missing {
				fmt.Fprintf(stdout, "  %s\n", n)
			}
			return 0
		}
		requested := make(map[string]bool, len(pkgs))
		for _, p := range pkgs {
			requested[p] = true
		}
		if err := mirror.AppendEntries(cfg.Allowlist, missing, requested); err != nil {
			fmt.Fprintf(stderr, "pac: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "added %d package(s) to the allowlist; review with git diff and push\n", len(missing))
		return 0

	case "remove":
		if len(pkgs) == 0 {
			fmt.Fprintln(stderr, "pac: mirror remove requires a package name")
			return 2
		}
		if err := validatePkgs(pkgs); err != nil {
			fmt.Fprintf(stderr, "pac: %v\n", err)
			return 2
		}
		removable, kept, err := mirror.Orphaned(r, resolverPrefix(cfg.BuilderImage), cfg.Allowlist, pkgs)
		if err != nil {
			fmt.Fprintf(stderr, "pac: aur closure resolution failed: %v\n", err)
			return 1
		}
		for _, k := range kept {
			fmt.Fprintf(stdout, "keep: %s -- still required by another approved package\n", k)
		}
		if len(removable) == 0 {
			fmt.Fprintln(stdout, "nothing removable")
			return 0
		}
		if err := mirror.RemoveEntries(cfg.Allowlist, removable); err != nil {
			fmt.Fprintf(stderr, "pac: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "removed %d package(s) from the allowlist; review with git diff and push\n", len(removable))
		return 0

	default:
		fmt.Fprintf(stderr, "pac: unknown mirror subcommand %q (add|remove|show|list)\n", sub)
		return 2
	}
}
