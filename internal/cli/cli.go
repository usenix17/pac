// Package cli parses arguments (including pacman-style aliases) and dispatches.
package cli

import (
	"fmt"
	"io"

	"starnix.net/pac/internal/cmd"
	"starnix.net/pac/internal/query"
	"starnix.net/pac/internal/run"
)

// Version is the pac version string.
const Version = "0.1.0"

const usage = `pac -- one front door for pacman + flatpak

Usage:
  pac update            update everything (alias: -Syu)
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
	case "-Ss":
		return append([]string{"search"}, args[1:]...)
	default:
		return args
	}
}

// Run executes pac with the given args and returns a process exit code.
// out receives user-facing messages; r is the subprocess runner.
func Run(args []string, r run.Runner, out io.Writer) int {
	args = normalize(args)

	if len(args) == 0 {
		fmt.Fprint(out, usage)
		return 0
	}
	switch args[0] {
	case "--help", "help":
		fmt.Fprint(out, usage)
		return 0
	case "--version":
		fmt.Fprintf(out, "pac %s\n", Version)
		return 0
	case "update":
		if err := cmd.Update(r); err != nil {
			fmt.Fprintf(out, "pac: update failed: %v\n", err)
			return 1
		}
		return 0
	case "search":
		if len(args) < 2 {
			fmt.Fprintln(out, "usage: pac search <term>")
			return 2
		}
		fmt.Fprint(out, query.Format(query.Search(r, args[1])))
		return 0
	default:
		fmt.Fprintf(out, "pac: unknown command %q (try `pac --help`)\n", args[0])
		return 2
	}
}
