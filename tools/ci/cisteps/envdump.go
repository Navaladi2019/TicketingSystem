package main

// The environment dump (TICK-1167 merge condition 2). `make SHELL=<this
// binary> <target>` runs each recipe line of the real goal through this
// binary instead of a shell. It records the arguments and the environment it
// was given, runs nothing, and exits 0. That is the environment and the
// command `make <target>` gives the scan, with every goal-conditional
// export, target-specific variable or extra prerequisite the makefile adds
// for that goal.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

const (
	// envDumpVar, set to 1 on make's command line, turns this binary into the
	// dump. A makefile that unexports it makes the recipe fail, which fails
	// the check.
	envDumpVar = "CISTEPS_ENV_DUMP"
	// envDumpPrefix starts the one line the dump prints per call.
	envDumpPrefix = "cisteps-env-dump "
)

// envRecord is one call of the dump: one recipe line make ran.
type envRecord struct {
	Args []string `json:"args"`
	Env  []string `json:"env"`
}

// envDumpRequested is whether this process is make's SHELL for the dump.
func envDumpRequested() bool { return os.Getenv(envDumpVar) == "1" }

// envDump prints this process's record to w.
func envDump(w io.Writer) int {
	b, err := json.Marshal(envRecord{Args: os.Args[1:], Env: os.Environ()})
	if err != nil {
		return 2
	}
	if _, err := fmt.Fprintln(w, envDumpPrefix+string(b)); err != nil {
		return 2
	}
	return 0
}

// envCaptured is whether make runs every recipe line through a SHELL given on
// its command line. GNU make on Windows runs a line with no shell
// metacharacter itself, so there the check reads only the database and
// Linux CI checks the environment.
func envCaptured() bool { return runtime.GOOS != "windows" }

// envDumpShell is the path make is given as SHELL: this binary.
func envDumpShell() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return exe, nil
}

// parseEnvDump splits make's output into the dump's records and every other
// line.
func parseEnvDump(out string) ([]envRecord, []string, error) {
	var recs []envRecord
	var other []string
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 1<<16), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		js, ok := strings.CutPrefix(line, envDumpPrefix)
		if !ok {
			if strings.TrimSpace(line) != "" {
				other = append(other, line)
			}
			continue
		}
		var r envRecord
		if err := json.Unmarshal([]byte(js), &r); err != nil {
			return nil, nil, fmt.Errorf("a garbled environment record: %w", err)
		}
		recs = append(recs, r)
	}
	return recs, other, sc.Err()
}

// envMap is env as a map; a variable listed twice keeps its last value.
func envMap(env []string) map[string]string {
	m := map[string]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		m[k] = v
	}
	return m
}
