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
	// -y, not --noninteractive: the latter suppresses flatpak's percentage
	// progress, so RunBar's bar would have nothing to render (see
	// installFlatpak). -y auto-confirms while keeping the progress stream.
	return r.RunBar("flatpak", "update", "-y")
}
