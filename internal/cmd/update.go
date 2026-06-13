// Package cmd holds one function per user-facing pac action.
package cmd

import "starnix.net/pac/internal/run"

// Update upgrades the whole system: pacman (official + aur-mirror) first so the
// base system moves before app runtimes, then Flatpak apps. If pacman fails we
// stop rather than update flatpaks against a half-updated base.
func Update(r run.Runner) error {
	if err := r.Run("sudo", "pacman", "-Syu"); err != nil {
		return err
	}
	return r.RunBar("flatpak", "update", "--noninteractive")
}
