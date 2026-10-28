package main

import (
	"bytes"
	"go/build/constraint"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The fixtures are modules of their own: stack has an untagged file, tests
// behind stacktag, !stacktag && go1.22 and an OS list, and a test behind
// stacktag && accept. In broken that last file redeclares a stacktag test
// helper, the shape of TICK-286 finding 4. nested/inner is a second module
// inside the first, behind innertag.

func TestDiscoverFixture(t *testing.T) {
	got, err := discover("testdata/ok")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"stacktag":        {"stack/stack_test.go"},
		"accept,stacktag": {"stack/accept_test.go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discover = %v, want %v", got, want)
	}
}

func TestModulesFindsNestedModules(t *testing.T) {
	got, err := modules("testdata/ok")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{".", "nested/inner"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("modules = %v, want %v", got, want)
	}
}

func TestBrokenTaggedFileFails(t *testing.T) {
	t.Setenv("GOFLAGS", "")
	var out bytes.Buffer
	code, err := run("testdata/broken", false, &out, goVet)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("exit %d, want 1\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "postRaw redeclared") {
		t.Errorf("the compile error is not in the output:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "FAIL: 1 of 3 tag sets: .:accept,stacktag\n") {
		t.Errorf("want only accept,stacktag in . to fail:\n%s", out.String())
	}
}

func TestFixtureThatCompilesPasses(t *testing.T) {
	t.Setenv("GOFLAGS", "")
	var out bytes.Buffer
	code, err := run("testdata/ok", false, &out, goVet)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit %d, want 0\n%s", code, out.String())
	}
}

func TestEveryFailingSetIsReported(t *testing.T) {
	var vetted []string
	vet := func(dir, tags string, _ io.Writer) error {
		vetted = append(vetted, filepath.ToSlash(dir)+":"+tags)
		return io.ErrUnexpectedEOF
	}
	var out bytes.Buffer
	code, err := run("testdata/ok", false, &out, vet)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"testdata/ok:accept,stacktag", "testdata/ok:stacktag", "testdata/ok/nested/inner:innertag"}
	if code != 1 || !reflect.DeepEqual(vetted, want) {
		t.Fatalf("exit %d, vetted %v, want %v: a failing set must not stop the others\n%s", code, vetted, want, out.String())
	}
}

func TestMinimalSets(t *testing.T) {
	cases := []struct {
		line string
		want [][]string
	}{
		{"//go:build integration", [][]string{{"integration"}}},
		{"//go:build identitystack && acceptance", [][]string{{"acceptance", "identitystack"}}},
		// Each side of an OR is vetted, so code reached only through b compiles too.
		{"//go:build a || b", [][]string{{"a"}, {"b"}}},
		{"//go:build (a && b) || c", [][]string{{"c"}, {"a", "b"}}},
		{"//go:build !relaytest", [][]string{nil}},
		{"//go:build go1.22", [][]string{nil}},
		{"//go:build linux && temporalstack", [][]string{{"temporalstack"}}},
		{"//go:build windows && spike", nil},
		{"//go:build ignore", nil},
		{"//go:build x && !x", nil},
	}
	for _, c := range cases {
		expr, err := constraint.Parse(c.line)
		if err != nil {
			t.Fatal(err)
		}
		got, err := minimalSets(expr)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, %v; want %v", c.line, got, err, c.want)
		}
	}
}

// Past maxTags a file is an error, not silently skipped.
func TestTooManyTagsIsAnError(t *testing.T) {
	var tags []string
	for i := 0; i <= maxTags; i++ {
		tags = append(tags, "t"+string(rune('a'+i)))
	}
	expr, err := constraint.Parse("//go:build " + strings.Join(tags, " && "))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := minimalSets(expr); err == nil {
		t.Fatalf("%d tags: want an error", len(tags))
	}
}

// The tag set that hid TICK-286 finding 4 must stay discovered in the real
// module, and the temporal-server module must stay found.
func TestRepositoryCoversTheAcceptanceTag(t *testing.T) {
	sets, err := discover("../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"acceptance,identitystack", "identitystack", "integration", "temporalstack"} {
		if len(sets[k]) == 0 {
			t.Errorf("tag set %q is not discovered; sets: %v", k, sets)
		}
	}
	mods, err := modules("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(" "+strings.Join(mods, " ")+" ", " temporal-server ") {
		t.Errorf("temporal-server is not among the modules %v", mods)
	}
}
