// Command racelock runs a command while holding a host-wide lock: by default
// the race-detector lock (TICK-153 D3), since every checkout and agent on one
// host shares the memory that `go test -race` needs, and two at once ran the
// host out of memory. With -slots it is a semaphore instead: the compose stack
// slots (TICK-153 D4), at most N stacks at once, the rest queued.
//
// Each slot is a listener on a loopback port (-port, -port+1, ...), so it
// needs no lock file, works the same on Linux, macOS and Windows, and is
// released by the OS when the holder exits or is killed. A run that needs
// -take slots holds all of them or none, so two waiting runs cannot each hold
// part of what the other needs. <env>_PORT overrides -port; <env>=off runs the
// command without the lock (-env, default NAVA_RACE_LOCK).
//
// Usage: racelock [-port 47613] [-slots 1] [-take 1] [-env NAVA_RACE_LOCK]
// [-what "go test -race"] -- <command> [args...]. It exits with the command's
// status.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	port := flag.Int("port", 47613, "first slot's loopback port")
	slots := flag.Int("slots", 1, "slots on the host")
	take := flag.Int("take", 1, "slots this command needs")
	env := flag.String("env", "NAVA_RACE_LOCK", "environment prefix: <env>=off skips the lock, <env>_PORT sets -port")
	what := flag.String("what", "go test -race", "what the slots are for, in the waiting message")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 || *take < 1 || *take > *slots {
		fmt.Fprintln(os.Stderr, "usage: racelock [-port n] [-slots n] [-take n] [-env NAME] [-what text] -- <command> [args...]")
		os.Exit(2)
	}
	var held []net.Listener
	if os.Getenv(*env) != "off" {
		if p := os.Getenv(*env + "_PORT"); p != "" {
			n, err := strconv.Atoi(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "racelock: %s_PORT=%q is not a port\n", *env, p)
				os.Exit(2)
			}
			*port = n
		}
		held = acquire(*port, *slots, *take, 30*time.Second, func(waited time.Duration) {
			fmt.Fprintf(os.Stderr, "racelock: waiting %s for %d of %d %s slots on this host (ports %d-%d)\n",
				waited.Round(time.Second), *take, *slots, *what, *port, *port+*slots-1)
		})
	}
	code := run(args)
	for _, l := range held {
		_ = l.Close()
	}
	os.Exit(code)
}

// acquire listens on take of the slots ports from port, retrying while
// fewer are free, and calls report every interval it has waited.
func acquire(port, slots, take int, interval time.Duration, report func(time.Duration)) []net.Listener {
	start := time.Now()
	next := interval
	for {
		var held []net.Listener
		for i := 0; i < slots && len(held) < take; i++ {
			l, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:"+strconv.Itoa(port+i))
			if err == nil {
				held = append(held, l)
			}
		}
		if len(held) == take {
			if waited := time.Since(start); waited >= time.Second {
				fmt.Fprintf(os.Stderr, "racelock: acquired after %s\n", waited.Round(time.Second))
			}
			return held
		}
		for _, l := range held {
			_ = l.Close()
		}
		if waited := time.Since(start); waited >= next {
			report(waited)
			next += interval
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func run(args []string) int {
	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...) //nolint:gosec // running the given command is the tool's job
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.ExitCode()
	default:
		fmt.Fprintf(os.Stderr, "racelock: %s: %v\n", strings.Join(args, " "), err)
		return 1
	}
}
