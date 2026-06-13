// Package query searches across pacman (official + aur-mirror) and flatpak,
// parsing each backend's output into a common Result.
package query

// Result is one search hit from any backend.
type Result struct {
	Source    string // pacman repo name (e.g. "extra", "aur-mirror") or "flatpak"
	Name      string // pacman pkgname, or flatpak application id
	Version   string
	Desc      string
	Installed bool // pacman only; flatpak search does not report install state
}
