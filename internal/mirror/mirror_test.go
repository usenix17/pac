package mirror_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestAppendEntries(t *testing.T) {
	path := writeAllowlist(t, sampleAllowlist)
	requested := map[string]bool{"phantomjs": true} // phantomjs explicit, qt5-doc a dep
	if err := mirror.AppendEntries(path, []string{"phantomjs", "qt5-doc"}, requested); err != nil {
		t.Fatalf("AppendEntries: %v", err)
	}

	// The new names are now approved, and the file still parses to the union.
	names, err := mirror.ApprovedNames(path)
	if err != nil {
		t.Fatalf("ApprovedNames: %v", err)
	}
	want := []string{"discord", "phantomjs", "qt5-doc", "qt5-webkit"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names after append = %v, want %v", names, want)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(body)
	if !strings.Contains(s, "      - name: phantomjs\n        approved: true\n        note: explicit\n") {
		t.Fatalf("phantomjs not appended as explicit:\n%s", s)
	}
	if !strings.Contains(s, "      - name: qt5-doc\n        approved: true\n        note: dependency\n") {
		t.Fatalf("qt5-doc not appended as dependency:\n%s", s)
	}
}

func TestExplicitNames(t *testing.T) {
	got, err := mirror.ExplicitNames(writeAllowlist(t, sampleAllowlist))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// only discord is note: explicit; qt5-webkit is note: dependency
	if !reflect.DeepEqual(got, []string{"discord"}) {
		t.Fatalf("ExplicitNames = %v, want [discord]", got)
	}
}

func TestRemoveEntries(t *testing.T) {
	path := writeAllowlist(t, sampleAllowlist)
	if err := mirror.RemoveEntries(path, []string{"qt5-webkit"}); err != nil {
		t.Fatalf("RemoveEntries: %v", err)
	}
	names, err := mirror.ApprovedNames(path)
	if err != nil {
		t.Fatalf("ApprovedNames: %v", err)
	}
	if !reflect.DeepEqual(names, []string{"discord"}) {
		t.Fatalf("names after remove = %v, want [discord]", names)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(body), "qt5-webkit") {
		t.Fatalf("qt5-webkit block not fully removed:\n%s", body)
	}
}
