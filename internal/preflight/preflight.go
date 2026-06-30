// Package preflight holds non-fatal, defense-in-depth self-checks run before
// privileged operations. The current check inspects pacman.conf to confirm the
// aur-mirror repo enforces signature verification and to flag repos that look
// like public/binary AUR sources.
package preflight

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// DefaultPacmanConf is the standard pacman configuration path.
const DefaultPacmanConf = "/etc/pacman.conf"

// officialRepos are the Arch sync repos that are expected and not flagged.
var officialRepos = map[string]bool{
	"core": true, "extra": true, "multilib": true,
	"core-testing": true, "extra-testing": true, "multilib-testing": true,
	"testing": true, "community": true, "community-testing": true,
	"core-debug": true, "extra-debug": true, "multilib-debug": true,
}

// SigLevelCheck parses the pacman config at path and writes non-fatal warnings
// to w. It warns when an existing [aur-mirror] stanza does not effectively
// require package signatures (Optional / TrustAll / Never / unset), and when a
// configured repo looks like a public or binary AUR source. A missing or
// unreadable config is silently ignored -- this is advisory only.
func SigLevelCheck(path string, w io.Writer) {
	f, err := os.Open(path)
	if err != nil {
		return // advisory only; nothing to check
	}
	defer f.Close()

	var (
		section    string
		globalSig  string
		mirrorSeen bool
		mirrorSig  string
		others     []string // non-official, non-mirror repo names
	)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			switch {
			case section == "aur-mirror":
				mirrorSeen = true
			case section != "options" && !officialRepos[section]:
				others = append(others, section)
			}
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != "SigLevel" {
			continue
		}
		v = strings.TrimSpace(v)
		switch section {
		case "options":
			globalSig = v
		case "aur-mirror":
			mirrorSig = v
		}
	}
	if err := sc.Err(); err != nil {
		return // best-effort; don't block the operation on a parse hiccup
	}

	if mirrorSeen {
		effective := mirrorSig
		if effective == "" {
			effective = globalSig // repo inherits [options] when it has no own line
		}
		if !packageSigRequired(effective) {
			shown := effective
			if shown == "" {
				shown = "(unset)"
			}
			fmt.Fprintf(w, "pac: WARNING: [aur-mirror] SigLevel is %q, not 'Required'.\n", shown)
			fmt.Fprintln(w, "pac:          Unsigned or untrusted AUR builds could be installed. Set 'SigLevel = Required' for [aur-mirror] in "+DefaultPacmanConf+".")
		}
	}
	for _, repo := range others {
		if looksLikePublicAUR(repo) {
			fmt.Fprintf(w, "pac: WARNING: repo [%s] looks like a public/binary AUR source.\n", repo)
			fmt.Fprintln(w, "pac:          pac expects AUR packages only via the signed [aur-mirror]; review this repo in "+DefaultPacmanConf+".")
		}
	}
}

// packageSigRequired reports whether siglevel enforces package signatures from
// trusted keys. pacman applies tokens left to right; a later token overrides an
// earlier one. TrustAll defeats the point of Required (it accepts any key), so
// it is treated as not-required regardless of order.
func packageSigRequired(siglevel string) bool {
	req := false
	trustAll := false
	for _, tok := range strings.Fields(siglevel) {
		switch tok {
		case "Required", "PackageRequired":
			req = true
		case "Optional", "Never", "PackageOptional", "PackageNever":
			req = false
		case "TrustAll", "PackageTrustAll":
			trustAll = true
		case "TrustedOnly", "PackageTrustedOnly":
			trustAll = false
		}
	}
	return req && !trustAll
}

// looksLikePublicAUR reports whether a repo name resembles a public AUR helper
// or binary-AUR repository (which would bypass the signed aur-mirror trust
// path). "aur-mirror" itself is excluded by the caller.
func looksLikePublicAUR(repo string) bool {
	lower := strings.ToLower(repo)
	switch lower {
	case "chaotic-aur", "archlinuxcn", "alerque":
		return true
	}
	return strings.Contains(lower, "aur")
}
