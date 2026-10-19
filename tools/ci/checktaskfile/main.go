// Command checktaskfile fails when a Taskfile.yml task's commands differ from
// its Makefile twin (T05, ruled in TICK-89). The Makefile side is what
// `make -n` prints, so variables are expanded the way make expands them.
//
// Usage: checktaskfile [-dir dir] [target ...]; the default targets are
// build, test and lint. `test:short` and `default` are Taskfile-only.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

func main() {
	dir := flag.String("dir", ".", "directory holding Makefile and Taskfile.yml")
	flag.Parse()
	targets := flag.Args()
	if len(targets) == 0 {
		targets = []string{"build", "test", "lint"}
	}

	problems, err := check(*dir, targets, makeCommands)
	if err != nil {
		fmt.Fprintln(os.Stderr, "check-taskfile:", err)
		os.Exit(2)
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "check-taskfile:", p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
	fmt.Println("check-taskfile: ok:", strings.Join(targets, ", "))
}

// check compares each target's commands; makeCmds returns the Makefile side.
func check(dir string, targets []string, makeCmds func(dir, target string) ([]string, error)) ([]string, error) {
	tasks, err := taskCommands(filepath.Join(dir, "Taskfile.yml"))
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, t := range targets {
		want, err := makeCmds(dir, t)
		if err != nil {
			return nil, err
		}
		got, ok := tasks[t]
		if !ok {
			problems = append(problems, fmt.Sprintf("Taskfile.yml has no task %q", t))
			continue
		}
		if strings.Join(want, "\n") != strings.Join(got, "\n") {
			problems = append(problems, fmt.Sprintf("%s: Makefile runs %q, Taskfile.yml runs %q", t, want, got))
		}
	}
	return problems, nil
}

// makeCommands is what `make -n <target>` would run, one command per line.
// GOLANGCI_LINT and similar ?= variables take their Makefile defaults.
func makeCommands(dir, target string) ([]string, error) {
	cmd := exec.Command("make", "-s", "-n", "-C", dir, target) //nolint:gosec,noctx // a CLI run to completion
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GOLANGCI_LINT=") && !strings.HasPrefix(kv, "MAKEFLAGS=") && !strings.HasPrefix(kv, "MFLAGS=") {
			env = append(env, kv)
		}
	}
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("make -n %s: %w: %s", target, err, stderr.String())
	}
	var cmds []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if l := normalize(line); l != "" && !strings.HasPrefix(l, "make") {
			cmds = append(cmds, l)
		}
	}
	return cmds, nil
}

// taskCommands returns every task's commands from a Taskfile, normalized.
func taskCommands(path string) (map[string][]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a path the caller chose
	if err != nil {
		return nil, err
	}
	var tf struct {
		Tasks map[string]struct {
			Cmds []yaml.Node `yaml:"cmds"`
		} `yaml:"tasks"`
	}
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := map[string][]string{}
	for name, task := range tf.Tasks {
		var cmds []string
		for _, n := range task.Cmds {
			var s string
			switch n.Kind {
			case yaml.ScalarNode:
				s = n.Value
			case yaml.MappingNode:
				var m struct {
					Cmd string `yaml:"cmd"`
				}
				if err := n.Decode(&m); err != nil {
					return nil, err
				}
				s = m.Cmd
			}
			for line := range strings.SplitSeq(s, "\n") {
				if l := normalize(line); l != "" {
					cmds = append(cmds, l)
				}
			}
		}
		out[name] = cmds
	}
	return out, nil
}

// normalize drops the Taskfile's {{exeExt}} (empty off Windows, where make
// runs) and collapses whitespace.
func normalize(s string) string {
	s = strings.ReplaceAll(s, "{{exeExt}}", "")
	return strings.Join(strings.Fields(s), " ")
}
