package mirror_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"starnix.net/pac/internal/mirror"
	"starnix.net/pac/internal/run"
)

const sampleAllowlist = `apiVersion: v1
kind: ConfigMap
data:
  allowlist.yaml: |
    packages:
      - name: discord
        approved: true
        note: explicit
      - name: qt5-webkit
        approved: true
        note: dependency
`

func writeAllowlist(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "allowlist.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return p
}

func TestApprovedNames(t *testing.T) {
	got, err := mirror.ApprovedNames(writeAllowlist(t, sampleAllowlist))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"discord", "qt5-webkit"} // sorted
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ApprovedNames = %v, want %v", got, want)
	}
}

func TestApprovedNamesIgnoresConfigMapName(t *testing.T) {
	// The ConfigMap's own "name: aur-allowlist" (no "- " prefix) must not be
	// counted as a package.
	body := "metadata:\n  name: aur-allowlist\ndata:\n  allowlist.yaml: |\n    packages:\n      - name: foo\n        approved: true\n        note: explicit\n"
	got, err := mirror.ApprovedNames(writeAllowlist(t, body))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"foo"}) {
		t.Fatalf("ApprovedNames = %v, want [foo]", got)
	}
}

func TestClosureParsesAurDepends(t *testing.T) {
	// aur depends -n emits tab-separated "target<TAB>dep" pairs.
	out := "phantomjs\tphantomjs\nphantomjs\tqt5-webkit\nqt5-webkit\tqt5-doc\n"
	f := &run.Fake{Results: []run.Call{{Out: out}}}

	got, err := mirror.Closure(f, []string{"aur", "depends", "-n"}, []string{"phantomjs"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"phantomjs", "qt5-doc", "qt5-webkit"} // unique, sorted
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Closure = %v, want %v", got, want)
	}
	wantCall := [][]string{{"aur", "depends", "-n", "phantomjs"}}
	if !reflect.DeepEqual(f.Calls, wantCall) {
		t.Fatalf("calls = %v, want %v", f.Calls, wantCall)
	}
}

func TestClosureWithDockerResolver(t *testing.T) {
	f := &run.Fake{Results: []run.Call{{Out: "foo\tfoo\n"}}}
	resolver := []string{"docker", "run", "--rm", "img", "aur", "depends", "-n"}
	if _, err := mirror.Closure(f, resolver, []string{"foo"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := [][]string{{"docker", "run", "--rm", "img", "aur", "depends", "-n", "foo"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("calls = %v, want %v", f.Calls, want)
	}
}

func TestMissing(t *testing.T) {
	closure := []string{"phantomjs", "qt5-doc", "qt5-webkit"}
	approved := []string{"discord", "qt5-webkit"}
	got := mirror.Missing(closure, approved)
	want := []string{"phantomjs", "qt5-doc"} // closure order preserved, qt5-webkit dropped
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Missing = %v, want %v", got, want)
	}
}

func TestClosureEmptyResolverErrors(t *testing.T) {
	if _, err := mirror.Closure(&run.Fake{}, nil, []string{"foo"}); err == nil {
		t.Fatal("expected error for empty resolver")
	}
}
