package validate_test

import (
	"testing"

	"starnix.net/pac/internal/validate"
)

func TestPkgName(t *testing.T) {
	valid := []string{"firefox", "qt5-webkit", "gtk2+", "lib32-glibc", "python-pip", "a", "0ad", "name@thing", "x.y_z"}
	for _, n := range valid {
		if err := validate.PkgName(n); err != nil {
			t.Errorf("PkgName(%q) = %v, want nil", n, err)
		}
	}
	invalid := []string{"", "Firefox", "fire fox", "foo;rm -rf", "a/b", "pkg$(x)", "name\n", "café", "a|b"}
	for _, n := range invalid {
		if err := validate.PkgName(n); err == nil {
			t.Errorf("PkgName(%q) = nil, want error", n)
		}
	}
}

func TestFlatpakID(t *testing.T) {
	valid := []string{"com.discordapp.Discord", "org.mozilla.firefox", "io.github.App-Name", "a"}
	for _, id := range valid {
		if err := validate.FlatpakID(id); err != nil {
			t.Errorf("FlatpakID(%q) = %v, want nil", id, err)
		}
	}
	// '-' is legal in a flatpak id, so dash-bearing ids are accepted; only
	// out-of-charset bytes (space, ';', '/', '+', '@') are rejected.
	invalid := []string{"", "app id", "foo;bar", "a/b", "gtk2+", "x@y"}
	for _, id := range invalid {
		if err := validate.FlatpakID(id); err == nil {
			t.Errorf("FlatpakID(%q) = nil, want error", id)
		}
	}
}

func TestTargetAcceptsEither(t *testing.T) {
	// pkgname charset, flatpak-id charset (uppercase), and a leading-dash name
	// (charset-legal; the "--" separator at exec is what neutralizes the dash).
	for _, n := range []string{"firefox", "com.discordapp.Discord", "gtk2+", "-rf"} {
		if err := validate.Target(n); err != nil {
			t.Errorf("Target(%q) = %v, want nil", n, err)
		}
	}
	for _, n := range []string{"", "foo;rm", "a/b", "two words", "x$y"} {
		if err := validate.Target(n); err == nil {
			t.Errorf("Target(%q) = nil, want error", n)
		}
	}
}
