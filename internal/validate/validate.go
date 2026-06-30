// Package validate checks user- and resolver-supplied package identifiers
// against the charsets their backends accept, before they reach a privileged
// exec. This is defense in depth alongside the literal "--" end-of-options
// separator each call site inserts: the separator stops a name being parsed as
// a flag, while these charsets reject names carrying shell- or option-like
// metacharacters in the first place.
package validate

import (
	"fmt"
	"regexp"
)

// pkgNameRe is the Arch Linux pkgname charset: lowercase letters, digits, and
// the punctuation '@', '.', '_', '+', '-'. See PKGBUILD(5).
var pkgNameRe = regexp.MustCompile(`^[a-z0-9@._+-]+$`)

// flatpakIDRe is the Flatpak application id charset. App ids are reverse-DNS
// and (unlike pkgnames) routinely carry uppercase in their final element, e.g.
// "com.discordapp.Discord", so the charset is broader on case but narrower on
// punctuation ('.', '_', '-' only).
var flatpakIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// PkgName validates name against the Arch pkgname charset.
func PkgName(name string) error {
	if name == "" || !pkgNameRe.MatchString(name) {
		return fmt.Errorf("invalid package name %q (allowed: lowercase letters, digits, and @._+-)", name)
	}
	return nil
}

// FlatpakID validates id against the Flatpak application id charset.
func FlatpakID(id string) error {
	if id == "" || !flatpakIDRe.MatchString(id) {
		return fmt.Errorf("invalid flatpak app id %q (allowed: letters, digits, and ._-)", id)
	}
	return nil
}

// Target validates a user-supplied install/remove target. The same token is
// resolved against both pacman (pkgname) and flatpak (app id), so it is
// accepted if it satisfies either charset and rejected only if it matches
// neither.
func Target(name string) error {
	if PkgName(name) == nil || FlatpakID(name) == nil {
		return nil
	}
	return fmt.Errorf("invalid package name %q (allowed: letters, digits, and @._+-)", name)
}
