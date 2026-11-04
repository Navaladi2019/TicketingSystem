package main

import (
	"bytes"
	"errors"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func testNow() time.Time { return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC) }

// fakeDocker answers the two listings from containers and networks, and an
// rm of a name in rmFail with that error output. It records every rm.
type fakeDocker struct {
	containers, networks string
	listErr              map[string]error // by listing: "ps" or "network"
	rmFail               map[string]string
	removed              [][]string
}

func (f *fakeDocker) run(combined bool, args ...string) ([]byte, error) {
	switch {
	case !combined && args[0] == "ps":
		if !slices.Contains(args, "{{.Names}}"+fields) {
			return nil, errors.New("ps: wrong --format")
		}
		return []byte(f.containers), f.listErr["ps"]
	case !combined && args[0] == "network" && args[1] == "ls":
		if !slices.Contains(args, "{{.Name}}"+fields) {
			return nil, errors.New("network ls: wrong --format")
		}
		return []byte(f.networks), f.listErr["network"]
	case combined:
		f.removed = append(f.removed, args)
		if out, ok := f.rmFail[args[len(args)-1]]; ok {
			return []byte(out + "\n"), errors.New("exit status 1")
		}
		return nil, nil
	}
	return nil, errors.New("unexpected docker call: " + strings.Join(args, " "))
}

const (
	sevenHoursAgo = "2026-10-27 05:00:00 +0000 UTC"
	tenMinutesAgo = "2026-10-27 11:50:00 +0000 UTC"
)

func listing() *fakeDocker {
	return &fakeDocker{
		containers: "nava-it-pg-1\t" + sevenHoursAgo + "\t\n" +
			"nava-it-pg-2\t" + tenMinutesAgo + "\t\n" + // a run in progress
			"nava-core-1\t" + sevenHoursAgo + "\tnava\n",
		networks: "nava-it-net-1\t2026-10-27 05:00:00.123 +0000 UTC\t\n",
	}
}

func TestRunRemovesStaleContainersThenNetworks(t *testing.T) {
	d := listing()
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr, testNow(), d.run); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	want := [][]string{{"rm", "-f", "nava-it-pg-1"}, {"network", "rm", "nava-it-net-1"}}
	if !slices.EqualFunc(d.removed, want, slices.Equal) {
		t.Errorf("removed %q, want %q", d.removed, want)
	}
	wantOut := "itreap: removed container nava-it-pg-1 (7h0m0s old)\n" +
		"itreap: removed network nava-it-net-1 (7h0m0s old)\n" +
		"itreap: 2 of 4 integration resources older than 6h0m0s, 0 failed\n"
	if stdout.String() != wantOut {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), wantOut)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr: %s", stderr.String())
	}
}

func TestRunDryRunRemovesNothing(t *testing.T) {
	d := listing()
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-n", "-older", "5m"}, &stdout, &stderr, testNow(), d.run); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if len(d.removed) != 0 {
		t.Errorf("dry run removed %q", d.removed)
	}
	wantOut := "itreap: would remove container nava-it-pg-1 (7h0m0s old)\n" +
		"itreap: would remove container nava-it-pg-2 (10m0s old)\n" +
		"itreap: would remove network nava-it-net-1 (7h0m0s old)\n" +
		"itreap: 3 of 4 integration resources older than 5m0s, 0 failed\n"
	if stdout.String() != wantOut {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), wantOut)
	}
}

// A resource gone since the listing is no failure; any other rm error is,
// and the run exits 1 after trying the rest.
func TestRunFailsOnlyOnARealRemovalError(t *testing.T) {
	d := listing()
	d.containers += "nava-iso-pg-3\t" + sevenHoursAgo + "\t\n"
	d.rmFail = map[string]string{
		"nava-it-pg-1":  "Error response from daemon: No such container: nava-it-pg-1",
		"nava-iso-pg-3": "Error response from daemon: cannot remove container: device busy",
	}
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr, testNow(), d.run); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if len(d.removed) != 3 {
		t.Errorf("tried %q, want all three stale resources", d.removed)
	}
	wantOut := "itreap: container nava-it-pg-1 already removed\n" +
		"itreap: removed network nava-it-net-1 (7h0m0s old)\n" +
		"itreap: 3 of 5 integration resources older than 6h0m0s, 1 failed\n"
	if stdout.String() != wantOut {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), wantOut)
	}
	wantErr := "itreap: docker rm -f nava-iso-pg-3: exit status 1: Error response from daemon: cannot remove container: device busy\n"
	if stderr.String() != wantErr {
		t.Errorf("stderr:\n%q\nwant:\n%q", stderr.String(), wantErr)
	}
}

func TestRunRefusesBadInput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		docker func() *fakeDocker
		code   int
		stderr string // substring
	}{
		{"zero age", []string{"-older", "0s"}, listing, 2, "usage: itreap [-older 6h] [-n]\n"},
		{"negative age", []string{"-older", "-1h"}, listing, 2, "usage: itreap [-older 6h] [-n]\n"},
		{"unknown flag", []string{"-x"}, listing, 2, "flag provided but not defined: -x"},
		{"bad duration", []string{"-older", "soon"}, listing, 2, `invalid value "soon" for flag -older`},
		{"container listing fails", nil, func() *fakeDocker {
			d := listing()
			d.listErr = map[string]error{"ps": errors.New("Cannot connect to the Docker daemon")}
			return d
		}, 1, "itreap: docker ps -a --filter name=nava-it- --filter name=nava-iso- --format {{.Names}}"},
		{"network listing fails", nil, func() *fakeDocker {
			d := listing()
			d.listErr = map[string]error{"network": errors.New("Cannot connect to the Docker daemon")}
			return d
		}, 1, "itreap: docker network ls --filter name=nava-it-net- --filter name=nava-iso-net- --format {{.Name}}"},
		{"malformed listing", nil, func() *fakeDocker {
			d := listing()
			d.networks = "nava-it-net-1\tyesterday\t\n"
			return d
		}, 1, `itreap: network nava-it-net-1: created "yesterday"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.docker()
			var stdout, stderr bytes.Buffer
			if code := run(tc.args, &stdout, &stderr, testNow(), d.run); code != tc.code {
				t.Fatalf("exit %d, want %d; stderr: %s", code, tc.code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.stderr) {
				t.Errorf("stderr %q does not contain %q", stderr.String(), tc.stderr)
			}
			if len(d.removed) != 0 {
				t.Errorf("removed %q after a refusal", d.removed)
			}
		})
	}
}

func TestRunHelpExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-h"}, &stdout, &stderr, testNow(), listing().run); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "-older") {
		t.Errorf("usage %q does not mention -older", stderr.String())
	}
}

// execDocker runs the binary it is given; the go tool stands in for docker.
func TestExecDockerRunsTheBinary(t *testing.T) {
	run := execDocker("go")
	out, err := run(false, "env", "GOOS")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != runtime.GOOS {
		t.Errorf("stdout = %q, want %q", got, runtime.GOOS)
	}
	out, err = run(true, "no-such-subcommand")
	if err == nil {
		t.Fatal("want an error from an unknown subcommand")
	}
	if !strings.Contains(string(out), "no-such-subcommand") {
		t.Errorf("combined output %q does not carry stderr", out)
	}
	if _, err := execDocker("no-such-docker-binary")(false, "ps"); err == nil {
		t.Error("want an error for a missing binary")
	}
}
