package main

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testEnv is the -env prefix every cli test uses, so a NAVA_RACE_LOCK set on
// the host (off, or a _PORT) never changes what the test sees, and the real
// lock port (47613) is never bound: every test passes its own -port.
const testEnv = "RACELOCK_CLI_TEST"

func getenv(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

// cliAsync runs cli in the background and returns its exit code channel and
// stderr.
func cliAsync(args []string, vars map[string]string) (<-chan int, *bytes.Buffer) {
	var stderr bytes.Buffer
	done := make(chan int, 1)
	go func() { done <- cli(args, getenv(vars), &stderr) }()
	return done, &stderr
}

func waitCode(t *testing.T, done <-chan int, within time.Duration) int {
	t.Helper()
	select {
	case code := <-done:
		return code
	case <-time.After(within):
		t.Fatalf("cli did not return within %s", within)
		return 0
	}
}

// Bad invocations are refused with exit 2 and the usage line, before any
// slot is taken or any command runs.
func TestCLIUsageErrors(t *testing.T) {
	port := strconv.Itoa(freePorts(t, 1))
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"no command", []string{"-env", testEnv, "-port", port}, "usage: racelock"},
		{"take zero", []string{"-env", testEnv, "-port", port, "-take", "0", "go", "version"}, "usage: racelock"},
		{"take more than slots", []string{"-env", testEnv, "-port", port, "-slots", "2", "-take", "3", "go", "version"}, "usage: racelock"},
		{"unknown flag", []string{"-bogus", "go", "version"}, "flag provided but not defined: -bogus"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if code := cli(c.args, getenv(nil), &stderr); code != 2 {
				t.Errorf("exit %d, want 2", code)
			}
			if !strings.Contains(stderr.String(), c.want) {
				t.Errorf("stderr %q, want %q", stderr.String(), c.want)
			}
		})
	}
}

func TestCLIHelpExitsZero(t *testing.T) {
	var stderr bytes.Buffer
	if code := cli([]string{"-h"}, getenv(nil), &stderr); code != 0 {
		t.Errorf("-h: exit %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "-take") {
		t.Errorf("-h: stderr %q does not list the flags", stderr.String())
	}
}

func TestCLIBadPortVariable(t *testing.T) {
	var stderr bytes.Buffer
	code := cli([]string{"-env", testEnv, "go", "version"}, getenv(map[string]string{testEnv + "_PORT": "abc"}), &stderr)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if want := `racelock: ` + testEnv + `_PORT="abc" is not a port`; !strings.Contains(stderr.String(), want) {
		t.Errorf("stderr %q, want %q", stderr.String(), want)
	}
}

// The command's own exit status is cli's, with and without the lock, and the
// slot is free again once cli returns.
func TestCLIRunsCommandUnderLock(t *testing.T) {
	port := freePorts(t, 1)
	p := strconv.Itoa(port)
	if code := cli([]string{"-env", testEnv, "-port", p, "--", "go", "version"}, getenv(nil), new(bytes.Buffer)); code != 0 {
		t.Errorf("go version: exit %d, want 0", code)
	}
	if code := cli([]string{"-env", testEnv, "-port", p, "--", "go", "no-such-command"}, getenv(nil), new(bytes.Buffer)); code != 2 {
		t.Errorf("go no-such-command: exit %d, want the command's 2", code)
	}
	_ = hold(t, port).Close() // the slot was released: binding it again succeeds
}

// A held slot makes cli wait, and it runs the command once the holder lets go.
func TestCLIWaitsForHeldSlot(t *testing.T) {
	port := freePorts(t, 1)
	first := hold(t, port)
	done, _ := cliAsync([]string{"-env", testEnv, "-port", strconv.Itoa(port), "go", "version"}, nil)
	select {
	case code := <-done:
		_ = first.Close()
		t.Fatalf("returned %d while the slot was held", code)
	case <-time.After(time.Second):
	}
	_ = first.Close()
	if code := waitCode(t, done, 30*time.Second); code != 0 {
		t.Errorf("exit %d after the release, want 0", code)
	}
}

// <env>=off runs the command without the lock: a held slot does not block it.
func TestCLIOffSkipsLock(t *testing.T) {
	port := freePorts(t, 1)
	held := hold(t, port)
	defer held.Close()
	done, _ := cliAsync([]string{"-env", testEnv, "-port", strconv.Itoa(port), "go", "version"}, map[string]string{testEnv: "off"})
	if code := waitCode(t, done, 30*time.Second); code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
}

// <env>_PORT overrides -port: with the -port slot held, a free _PORT slot is
// taken at once.
func TestCLIPortVariableOverridesFlag(t *testing.T) {
	base := freePorts(t, 2)
	held := hold(t, base)
	defer held.Close()
	done, _ := cliAsync([]string{"-env", testEnv, "-port", strconv.Itoa(base), "go", "version"},
		map[string]string{testEnv + "_PORT": strconv.Itoa(base + 1)})
	if code := waitCode(t, done, 30*time.Second); code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
}
