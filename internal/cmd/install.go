package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"starnix.net/pac/internal/run"
	"starnix.net/pac/internal/validate"
)

// pacmanHas reports whether name is available in a pacman sync repo
// (official or aur-mirror): `pacman -Si` exits 0 only if it exists. The "--"
// guards against a name that begins with '-' being read as a flag.
func pacmanHas(r run.Runner, name string) bool {
	_, err := r.Capture("pacman", "-Si", "--", name)
	return err == nil
}

// flatpakAppID returns the Flathub application id for name, matching either the
// application id or the human-readable name (case-insensitive), and whether a
// match was found.
func flatpakAppID(r run.Runner, name string) (string, bool) {
	out, _ := r.Capture("flatpak", "search", "--", name)
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "\t")
		if len(f) < 4 {
			continue
		}
		human, appID := strings.TrimSpace(f[0]), strings.TrimSpace(f[2])
		if strings.EqualFold(appID, name) || strings.EqualFold(human, name) {
			return appID, true
		}
	}
	return "", false
}

// Install resolves name across pacman (official + aur-mirror) and flatpak and
// installs it. prefer ("system"|"flatpak"|"ask") decides when a package is
// available from both; "ask" prompts on stdin. Returns a process exit code.
func Install(r run.Runner, prefer, name string, stdin io.Reader, stdout, stderr io.Writer) int {
	if err := validate.Target(name); err != nil {
		fmt.Fprintf(stderr, "pac: %v\n", err)
		return 2
	}
	inPacman := pacmanHas(r, name)
	appID, inFlatpak := flatpakAppID(r, name)

	switch {
	case inPacman && inFlatpak:
		choice := prefer
		if prefer == "ask" {
			choice = promptChoice(name, appID, stdin, stdout)
		}
		if choice == "flatpak" {
			return installFlatpak(r, appID, stderr)
		}
		return installPacman(r, name, stderr)
	case inPacman:
		return installPacman(r, name, stderr)
	case inFlatpak:
		return installFlatpak(r, appID, stderr)
	default:
		fmt.Fprintf(stderr, "pac: %q is not in repos, aur-mirror, or flatpak.\n", name)
		fmt.Fprintf(stderr, "If it is an AUR package, add it to your mirror:\n  pac mirror add %s\n", name)
		return 1
	}
}

func installPacman(r run.Runner, name string, stderr io.Writer) int {
	if err := r.Run("sudo", "pacman", "-S", "--", name); err != nil {
		fmt.Fprintf(stderr, "pac: install failed: %v\n", err)
		return 1
	}
	return 0
}

func installFlatpak(r run.Runner, appID string, stderr io.Writer) int {
	// appID comes from flatpak's own search output; re-validate it before exec
	// as defense in depth, and pass "--" so it can never be read as a flag.
	if err := validate.FlatpakID(appID); err != nil {
		fmt.Fprintf(stderr, "pac: %v\n", err)
		return 1
	}
	// Use -y (assume yes), NOT --noninteractive: --noninteractive suppresses
	// flatpak's percentage progress, leaving RunBar's bar with nothing to
	// parse. -y auto-confirms but keeps the progress stream we render.
	if err := r.RunBar(appID, "flatpak", "install", "-y", "--", appID); err != nil {
		fmt.Fprintf(stderr, "pac: install failed: %v\n", err)
		return 1
	}
	return 0
}

// promptChoice asks whether to install from system or flatpak. Anything other
// than f/flatpak (including empty input) defaults to system.
func promptChoice(name, appID string, stdin io.Reader, stdout io.Writer) string {
	fmt.Fprintf(stdout, "%q is available from both sources:\n  [s] system (pacman/aur-mirror)\n  [f] flatpak (%s)\nInstall from? [s/f]: ", name, appID)
	sc := bufio.NewScanner(stdin)
	if sc.Scan() {
		switch strings.ToLower(strings.TrimSpace(sc.Text())) {
		case "f", "flatpak":
			return "flatpak"
		}
	}
	return "system"
}
