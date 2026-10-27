package main

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{
			name: "output events, the other actions dropped",
			in: `{"Action":"start","Package":"p"}
{"Action":"run","Package":"p","Test":"TestA"}
{"Action":"output","Package":"p","Test":"TestA","Output":"=== RUN   TestA\n"}
{"Action":"output","Package":"p","Test":"TestA","Output":"--- PASS: TestA (0.00s)\n"}
{"Action":"pass","Package":"p","Test":"TestA"}
{"Action":"output","Package":"p","Output":"ok  \tp\t0.1s\n"}
{"Action":"pass","Package":"p"}
`,
			want: "[g] === RUN   TestA\n[g] --- PASS: TestA (0.00s)\n[g] ok  \tp\t0.1s\n",
		},
		{
			name: "a line split over events is printed once, whole",
			in: `{"Action":"output","Package":"p","Test":"TestA","Output":"    a_test.go:9: hal"}
{"Action":"output","Package":"p","Test":"TestA","Output":"f a line\n    next\n"}
`,
			want: "[g]     a_test.go:9: half a line\n[g]     next\n",
		},
		{
			name: "a last line without a newline",
			in:   `{"Action":"output","Package":"p","Output":"exit status 2"}`,
			want: "[g] exit status 2\n",
		},
		{
			name: "not an event: go test's own errors",
			in:   "go: no such package\n{\"Action\":\"build-output\",\"ImportPath\":\"p\",\"Output\":\"p/a.go:1: syntax error\\n\"}\n{not json\n",
			want: "[g] go: no such package\n[g] p/a.go:1: syntax error\n[g] {not json\n",
		},
		{
			// What the timeout dump looks like under -json (Go 1.26), cut short.
			name: "timeout dump",
			in: `{"Action":"output","Package":"p","Test":"TestHang","Output":"panic: test timed out after 6m0s\n"}
{"Action":"output","Package":"p","Test":"TestHang","Output":"\trunning tests:\n\t\tTestHang (6m0s)\n"}
{"Action":"fail","Package":"p","Test":"TestHang"}
`,
			want: "[g] panic: test timed out after 6m0s\n[g] \trunning tests:\n[g] \t\tTestHang (6m0s)\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out strings.Builder
			if err := run(strings.NewReader(c.in), &out, "g"); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != c.want {
				t.Errorf("got\n%q\nwant\n%q", got, c.want)
			}
		})
	}
}

// Each line is one Write, so the groups' lines interleave whole.
func TestOneWritePerLine(t *testing.T) {
	var w writes
	in := `{"Action":"output","Output":"a\nb"}` + "\n" + `{"Action":"output","Output":"c\n"}` + "\n"
	if err := run(strings.NewReader(in), &w, "g"); err != nil {
		t.Fatal(err)
	}
	want := []string{"[g] a\n", "[g] bc\n"}
	if strings.Join(w, "|") != strings.Join(want, "|") {
		t.Errorf("writes %q, want %q", w, want)
	}
}

type writes []string

func (w *writes) Write(p []byte) (int, error) {
	*w = append(*w, string(p))
	return len(p), nil
}
