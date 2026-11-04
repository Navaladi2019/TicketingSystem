package main

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCLIUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"a", "b"}} {
		var stdout, stderr strings.Builder
		if code := cli(args, strings.NewReader(""), &stdout, &stderr); code != 2 {
			t.Errorf("args %q: exit %d, want 2", args, code)
		}
		if got := stderr.String(); got != "usage: go test -json ... | testlog <group>\n" {
			t.Errorf("args %q: stderr %q", args, got)
		}
		if stdout.Len() != 0 {
			t.Errorf("args %q: stdout %q, want nothing", args, stdout.String())
		}
	}
}

func TestCLIPrints(t *testing.T) {
	var stdout, stderr strings.Builder
	in := `{"Action":"output","Output":"ok\n"}` + "\n"
	if code := cli([]string{"unit"}, strings.NewReader(in), &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
	if got := stdout.String(); got != "[unit] ok\n" {
		t.Errorf("stdout %q", got)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr %q, want nothing", stderr.String())
	}
}

var errBoom = errors.New("boom")

// failAfter accepts n writes, then fails every one.
type failAfter struct {
	n      int
	writes []string
}

func (w *failAfter) Write(p []byte) (int, error) {
	if len(w.writes) >= w.n {
		return 0, errBoom
	}
	w.writes = append(w.writes, string(p))
	return len(p), nil
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errBoom }

// A failed write or read stops the run with that error, exit 1, and nothing
// after the failure is printed.
func TestErrors(t *testing.T) {
	for _, c := range []struct {
		name   string
		in     io.Reader
		okW    int
		writes string
	}{
		{"write of a test event", strings.NewReader(`{"Action":"output","Output":"a\n"}` + "\n" + `{"Action":"output","Output":"b\n"}` + "\n"), 1, "[g] a\n"},
		{"write of a plain line", strings.NewReader("go: oops\nnext\n"), 0, ""},
		{"second line of one event", strings.NewReader(`{"Action":"output","Output":"a\nb\n"}` + "\n"), 1, "[g] a\n"},
		{"flush of the last line", strings.NewReader(`{"Action":"output","Output":"tail"}`), 0, ""},
		{"read", errReader{}, 10, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := &failAfter{n: c.okW}
			var stderr strings.Builder
			if code := cli([]string{"g"}, c.in, w, &stderr); code != 1 {
				t.Fatalf("exit %d, want 1", code)
			}
			if got := stderr.String(); got != "testlog: boom\n" {
				t.Errorf("stderr %q", got)
			}
			if got := strings.Join(w.writes, ""); got != c.writes {
				t.Errorf("printed %q, want %q", got, c.writes)
			}
		})
	}
}

// A line longer than the scanner's 16 MiB limit is an error, not a silent
// truncation of the log.
func TestLineTooLong(t *testing.T) {
	var stdout, stderr strings.Builder
	in := strings.Repeat("x", 16*1024*1024+1)
	if code := cli([]string{"g"}, strings.NewReader(in), &stdout, &stderr); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "testlog: bufio.Scanner: token too long") {
		t.Errorf("stderr %q", stderr.String())
	}
}
