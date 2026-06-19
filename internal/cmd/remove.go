package cmd

import (
	"fmt"
	"io"

	"starnix.net/pac/internal/run"
)

// pacmanInstalled reports whether name is installed as a pacman package.
func pacmanInstalled(r run.Runner, name string) bool {
	_, err := r.Capture("pacman", "-Qi", name)
	return err == nil
}

// flatpakInstalled reports whether name is an installed flatpak.
func flatpakInstalled(r run.Runner, name string) bool {
	_, err := r.Capture("flatpak", "info", name)
	return err == nil
}

// Remove uninstalls name from whichever backend has it installed. pacman is
// checked first and wins if (rarely) both report it. Returns a process exit
// code.
func Remove(r run.Runner, name string, stderr io.Writer) int {
	switch {
	case pacmanInstalled(r, name):
		// -R removes just the named package, not its now-unneeded dependencies
		// (-Rs). This is deliberate: pac should not surprise-remove a chain of
		// dependencies; the user can run pacman -Rs manually for that.
		if err := r.Run("sudo", "pacman", "-R", name); err != nil {
			fmt.Fprintf(stderr, "pac: remove failed: %v\n", err)
			return 1
		}
		return 0
	case flatpakInstalled(r, name):
		// -y, not --noninteractive: keep the progress stream RunBar renders
		// (see installFlatpak). Uninstall rarely emits a percentage, but the
		// flag stays consistent and still auto-confirms.
		if err := r.RunBar(name, "flatpak", "uninstall", "-y", name); err != nil {
			fmt.Fprintf(stderr, "pac: remove failed: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "pac: %q is not installed (via pacman or flatpak).\n", name)
		return 1
	}
}
