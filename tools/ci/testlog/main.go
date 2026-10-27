// Command testlog prints a `go test -json` stream from stdin as go test -v
// would, every line prefixed with "[<group>] ", as the stream arrives
// (TICK-225). scripts/ci/test-integration.sh runs its test groups at the same
// time and pipes each through it, so a step cancelled at timeout-minutes still
// shows each group's output up to that point, and a timeout dump shows which
// group it came from. A line that is not a test event (go test's own errors)
// is printed as it is, with the prefix.
//
// Each line goes out in one write, so lines of groups printing at the same
// time do not mix within a line.
//
// Usage:
//
//	go test -json [flags] [packages] | testlog <group>
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go test -json ... | testlog <group>")
		os.Exit(2)
	}
	if err := run(os.Stdin, os.Stdout, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "testlog:", err)
		os.Exit(1)
	}
}

// event is the part of a `go test -json` line testlog prints.
type event struct {
	Action string
	Output string
}

func run(r io.Reader, w io.Writer, group string) error {
	p := printer{w: w, prefix: "[" + group + "] "}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		var e event
		if !strings.HasPrefix(line, "{") || json.Unmarshal([]byte(line), &e) != nil || e.Action == "" {
			if err := p.write(line + "\n"); err != nil {
				return err
			}
			continue
		}
		// output and build-output carry text; the other actions repeat what
		// the output already says (=== RUN, --- PASS).
		if err := p.write(e.Output); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return p.flush()
}

// printer cuts text into lines. test2json splits a long line, or one still
// unfinished when the next event arrives, over several events, so the rest of
// a line waits in partial for its newline.
type printer struct {
	w       io.Writer
	prefix  string
	partial string
}

func (p *printer) write(s string) error {
	s = p.partial + s
	p.partial = ""
	for s != "" {
		line, rest, ok := strings.Cut(s, "\n")
		if !ok {
			p.partial = line
			return nil
		}
		if _, err := io.WriteString(p.w, p.prefix+line+"\n"); err != nil {
			return err
		}
		s = rest
	}
	return nil
}

// flush prints a last line the stream ended without a newline.
func (p *printer) flush() error {
	if p.partial == "" {
		return nil
	}
	return p.write("\n")
}
