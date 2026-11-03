// Command itreap removes the Docker containers and networks that killed
// integration runs left behind (TICK-185): scripts/ci/test-integration.sh
// (nava-it-pg-<pid>[-s<n>], nava-it-pgb-pooled-<pid>, nava-it-pgb-noprep-<pid>,
// nava-it-net-<pid>), scripts/ci/isolation.sh (nava-iso-pg-<pid>,
// nava-iso-pgb-<pid>, nava-iso-net-<pid>) and the health integration test
// (nava-it-health-<pid>-<port>). Those scripts remove their own resources on
// every exit they can trap (<pid> is the pid plus a random suffix since
// TICK-182, still digits); only SIGKILL, or a host killing the process tree,
// leaves them, and every leftover holds one of Docker's address pools.
//
// Only a resource whose name has exactly one of those forms, that carries no
// compose project label and that is older than -older is removed, so the
// compose stacks (nava, nava-<id>, ci-isolated's projects) and a run still in
// progress are never touched. Containers go first, then the networks.
//
// Usage: itreap [-older 6h] [-n]. -n prints what it would remove.
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Resource is one Docker container or network as docker ls prints it.
type Resource struct {
	Kind    string // "container" or "network"
	Name    string
	Created time.Time
	Project string // com.docker.compose.project label, "" for none
}

var (
	containerName = regexp.MustCompile(`^nava-(it-(pg-[0-9]+(-s[0-9]+)?|pgb-(pooled|noprep)-[0-9]+|health-[0-9]+-[0-9]+)|iso-pgb?-[0-9]+)$`)
	networkName   = regexp.MustCompile(`^nava-(it|iso)-net-[0-9]+$`)
)

// stale returns the resources itreap removes at now: an integration name, no
// compose project, created more than older ago. Containers come before
// networks, since Docker refuses to remove a network a container is on.
func stale(rs []Resource, now time.Time, older time.Duration) []Resource {
	var cs, ns []Resource
	for _, r := range rs {
		if r.Project != "" || r.Created.IsZero() || now.Sub(r.Created) <= older {
			continue
		}
		switch {
		case r.Kind == "container" && containerName.MatchString(r.Name):
			cs = append(cs, r)
		case r.Kind == "network" && networkName.MatchString(r.Name):
			ns = append(ns, r)
		}
	}
	return append(cs, ns...)
}

// createdLayout is docker ls's CreatedAt; a network's has fractional seconds,
// which time.Parse accepts without them in the layout.
const createdLayout = "2006-01-02 15:04:05 -0700 MST"

// parse reads docker ls lines of name, CreatedAt and compose project,
// tab-separated.
func parse(kind string, out []byte) ([]Resource, error) {
	var rs []Resource
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			return nil, fmt.Errorf("%s line %q: want 3 tab-separated fields", kind, line)
		}
		t, err := time.Parse(createdLayout, f[1])
		if err != nil {
			return nil, fmt.Errorf("%s %s: created %q: %w", kind, f[0], f[1], err)
		}
		rs = append(rs, Resource{Kind: kind, Name: f[0], Created: t, Project: f[2]})
	}
	return rs, sc.Err()
}

// fields is the --format after the name: docker ps calls the name .Names,
// docker network ls .Name.
const fields = `{{"\t"}}{{.CreatedAt}}{{"\t"}}{{.Label "com.docker.compose.project"}}`

// dockerFunc runs docker with args: combined selects stdout and stderr
// together (for rm, whose error text gone reads), else stdout alone (for the
// listings). It is a parameter of run so tests need no Docker daemon.
type dockerFunc func(combined bool, args ...string) ([]byte, error)

// execDocker runs the docker binary bin.
func execDocker(bin string) dockerFunc {
	return func(combined bool, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(context.Background(), bin, args...) //nolint:gosec // docker with arguments from this file
		if combined {
			return cmd.CombinedOutput()
		}
		return cmd.Output()
	}
}

func list(docker dockerFunc, kind string, args ...string) ([]Resource, error) {
	out, err := docker(false, args...)
	if err != nil {
		return nil, fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}
	return parse(kind, out)
}

// gone reports whether docker's error output says the resource no longer
// exists ("No such container: x", "network x not found").
func gone(out []byte) bool {
	s := strings.ToLower(string(out))
	return strings.Contains(s, "no such container") || strings.Contains(s, "not found")
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, time.Now(), execDocker("docker"))) }

// run reaps with docker as of now and returns the exit code: 0, 1 when a
// listing or a removal failed, 2 for bad flags.
func run(args []string, stdout, stderr io.Writer, now time.Time, docker dockerFunc) int {
	fs := flag.NewFlagSet("itreap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	older := fs.Duration("older", 6*time.Hour, "remove only resources created longer ago than this")
	dry := fs.Bool("n", false, "print what would be removed, remove nothing")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // as flag.ExitOnError does
		}
		return 2
	}
	if *older <= 0 {
		fmt.Fprintln(stderr, "usage: itreap [-older 6h] [-n]")
		return 2
	}
	// The name filters are substring matches; stale applies the exact forms.
	cs, err := list(docker, "container", "ps", "-a", "--filter", "name=nava-it-", "--filter", "name=nava-iso-",
		"--format", "{{.Names}}"+fields)
	if err != nil {
		fmt.Fprintln(stderr, "itreap:", err)
		return 1
	}
	ns, err := list(docker, "network", "network", "ls", "--filter", "name=nava-it-net-", "--filter", "name=nava-iso-net-",
		"--format", "{{.Name}}"+fields)
	if err != nil {
		fmt.Fprintln(stderr, "itreap:", err)
		return 1
	}
	failed := 0
	old := stale(append(cs, ns...), now, *older)
	for _, r := range old {
		age := now.Sub(r.Created).Round(time.Minute)
		if *dry {
			fmt.Fprintf(stdout, "itreap: would remove %s %s (%s old)\n", r.Kind, r.Name, age)
			continue
		}
		rm := []string{"rm", "-f", r.Name}
		if r.Kind == "network" {
			rm = []string{"network", "rm", r.Name}
		}
		out, err := docker(true, rm...)
		switch {
		case err != nil && gone(out):
			// Removed since the listing: by its own run, or by another reaper.
			fmt.Fprintf(stdout, "itreap: %s %s already removed\n", r.Kind, r.Name)
		case err != nil:
			fmt.Fprintf(stderr, "itreap: docker %s: %v: %s\n", strings.Join(rm, " "), err, bytes.TrimSpace(out))
			failed++
		default:
			fmt.Fprintf(stdout, "itreap: removed %s %s (%s old)\n", r.Kind, r.Name, age)
		}
	}
	fmt.Fprintf(stdout, "itreap: %d of %d integration resources older than %s, %d failed\n", len(old), len(cs)+len(ns), *older, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
