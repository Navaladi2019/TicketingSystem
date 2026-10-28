package main

import (
	"bytes"
	"go/build/constraint"
	"io"
	"reflect"
	"strings"
	"testing"
)

// The fixtures are modules of their own: stack has an untagged file, tests
// behind stacktag, !stacktag && go1.22 and an OS list, and a test behind
// stacktag && accept. In broken that last file redeclares a stacktag test
// helper, the shape of TICK-286 finding 4. nested/inner is a module inside the
// module, behind a tag buildtags must not see.

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
	if !strings.Contains(out.String(), "FAIL: 1 of 2 tag sets: accept,stacktag\n") {
		t.Errorf("want only accept,stacktag to fail:\n%s", out.String())
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
	vet := func(_, tags string, _ io.Writer) error {
		vetted = append(vetted, tags)
		return io.ErrUnexpectedEOF
	}
	var out bytes.Buffer
	code, err := run("testdata/ok", false, &out, vet)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || !reflect.DeepEqual(vetted, []string{"accept,stacktag", "stacktag"}) {
		t.Fatalf("exit %d, vetted %v: a failing set must not stop the others\n%s", code, vetted, out.String())
	}
}

func TestSmallestSet(t *testing.T) {
	cases := []struct {
		line string
		want []string
		ok   bool
	}{
		{"//go:build integration", []string{"integration"}, true},
		{"//go:build identitystack && acceptance", []string{"acceptance", "identitystack"}, true},
		{"//go:build a || b", []string{"a"}, true},
		{"//go:build !relaytest", nil, true},
		{"//go:build go1.22", nil, true},
		{"//go:build linux && temporalstack", []string{"temporalstack"}, true},
		{"//go:build windows && spike", nil, false},
		{"//go:build ignore", nil, false},
		{"//go:build x && !x", nil, false},
	}
	for _, c := range cases {
		expr, err := constraint.Parse(c.line)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := smallestSet(expr)
		if ok != c.ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, %v; want %v, %v", c.line, got, ok, c.want, c.ok)
		}
	}
}

// The tag set that hid TICK-286 finding 4 must stay discovered in the real
// module, so the step cannot quietly stop covering it.
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
}
