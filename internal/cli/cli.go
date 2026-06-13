// Package cli parses arguments (including pacman-style aliases) and dispatches.
package cli

import (
	"fmt"
	"io"
	"strings"

	"starnix.net/pac/internal/cmd"
	"starnix.net/pac/internal/config"
	"starnix.net/pac/internal/query"
	"starnix.net/pac/internal/run"
)

// Version is the pac version string.
const Version = "0.1.0"

const usage = `pac -- one front door for pacman + flatpak

Usage:
  pac update            update everything (alias: -Syu)
  pac install <name>    install from repos, aur-mirror, or flatpak (alias: -S)
  pac remove <name>     remove an installed package (alias: -R)
  pac search <term>     search repos + aur-mirror + flatpak (alias: -Ss)
  pac --version         print version
  pac --help            show this help
`

// normalize translates pacman-style flag aliases into subcommands.
func normalize(args []string) []string {
	if len(args) == 0 {
		return args
	}
	switch args[0] {
	case "-Syu":
		return append([]string{"update"}, args[1:]...)
	case "-S":
		return append([]string{"install"}, args[1:]...)
	case "-R":
		return append([]string{"remove"}, args[1:]...)
	case "-Ss":
		return append([]string{"search"}, args[1:]...)
	default:
		return args
	}
}

// Run executes pac with the given args and returns a process exit code.
// Normal output goes to stdout; errors go to stderr; stdin is read only for
// interactive prompts (install disambiguation). r is the subprocess runner.
func Run(args []string, r run.Runner, stdin io.Reader, stdout, stderr io.Writer) int {
	args = normalize(args)

	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}
	switch args[0] {
	case "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	case "--version":
		fmt.Fprintf(stdout, "pac %s\n", Version)
		return 0
	case "update":
		if err := cmd.Update(r); err != nil {
			fmt.Fprintf(stderr, "pac: update failed: %v\n", err)
			return 1
		}
		return 0
	case "install":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "pac: install requires a package name (usage: pac install <name>)")
			return 2
		}
		return cmd.Install(r, config.Load().Prefer, args[1], stdin, stdout, stderr)
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "pac: remove requires a package name (usage: pac remove <name>)")
			return 2
		}
		return cmd.Remove(r, args[1], stderr)
	case "search":
		term := strings.TrimSpace(strings.Join(args[1:], " "))
		if term == "" {
			fmt.Fprintln(stderr, "pac: search requires a term (usage: pac search <term>)")
			return 2
		}
		fmt.Fprint(stdout, query.Format(query.Search(r, term)))
		return 0
	default:
		fmt.Fprintf(stderr, "pac: unknown command %q (try `pac --help`)\n", args[0])
		return 2
	}
}
