// Command racelock runs a command while holding the host-wide race-detector
// lock (TICK-153 D3): every checkout and agent on one host shares the memory
// that `go test -race` needs, and two at once ran the host out of memory.
//
// The lock is a listener on a fixed loopback port, so it needs no lock file,
// works the same on Linux, macOS and Windows, and is released by the OS when
// the holder exits or is killed. NAVA_RACE_LOCK_PORT overrides the port;
// NAVA_RACE_LOCK=off runs the command without the lock.
//
// Usage: racelock -- <command> [args...]. It exits with the command's status.
package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultPort = "47613"

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: racelock -- <command> [args...]")
		os.Exit(2)
	}
	var l net.Listener
	if os.Getenv("NAVA_RACE_LOCK") != "off" {
		port := os.Getenv("NAVA_RACE_LOCK_PORT")
		if port == "" {
			port = defaultPort
		}
		l = acquire("127.0.0.1:"+port, 30*time.Second, func(waited time.Duration) {
			fmt.Fprintf(os.Stderr, "racelock: waiting %s for another go test -race on this host (port %s)\n",
				waited.Round(time.Second), port)
		})
	}
	code := run(args)
	if l != nil {
		_ = l.Close()
	}
	os.Exit(code)
}

// acquire listens on addr, retrying while another holder has it, and calls
// report every interval it has waited.
func acquire(addr string, interval time.Duration, report func(time.Duration)) net.Listener {
	start := time.Now()
	next := interval
	for {
		l, err := net.Listen("tcp", addr)
		if err == nil {
			if waited := time.Since(start); waited >= time.Second {
				fmt.Fprintf(os.Stderr, "racelock: acquired after %s\n", waited.Round(time.Second))
			}
			return l
		}
		if waited := time.Since(start); waited >= next {
			report(waited)
			next += interval
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func run(args []string) int {
	cmd := exec.Command(args[0], args[1:]...)
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
