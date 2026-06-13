package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"starnix.net/pac/internal/cmd"
	"starnix.net/pac/internal/config"
	"starnix.net/pac/internal/run"
)

const mirrorAllowlist = "data:\n  allowlist.yaml: |\n    packages:\n      - name: discord\n        approved: true\n        note: explicit\n"

func mirrorCfg(t *testing.T) config.Config {
	t.Helper()
	p := filepath.Join(t.TempDir(), "allowlist.yaml")
	if err := os.WriteFile(p, []byte(mirrorAllowlist), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return config.Config{Allowlist: p, BuilderImage: "img"}
}

func TestMirrorList(t *testing.T) {
	var out bytes.Buffer
	code := cmd.Mirror(&run.Fake{}, mirrorCfg(t), []string{"list"}, &out, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "discord") {
		t.Fatalf("list output missing discord: %q", out.String())
	}
}

func TestMirrorNoSubcommandExits2(t *testing.T) {
	code := cmd.Mirror(&run.Fake{}, mirrorCfg(t), nil, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestMirrorUnknownSubcommandExits2(t *testing.T) {
	code := cmd.Mirror(&run.Fake{}, mirrorCfg(t), []string{"frob"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestMirrorAddMissingPkgExits2(t *testing.T) {
	code := cmd.Mirror(&run.Fake{}, mirrorCfg(t), []string{"add"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}
