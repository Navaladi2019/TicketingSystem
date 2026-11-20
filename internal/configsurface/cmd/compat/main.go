// Command compat is check 5 of the upgrade-compatibility suite (ADR-008 X2,
// M-X2; PLAN-009 P9-02): it fails, naming each item, when an item listed in
// the base ref's internal/configsurface/testdata/inventory.json is missing
// from HEAD's. That catches an item removed from both the code and the
// inventory, which checks 1 to 4 cannot see.
//
// A removal is allowed only when HEAD's major is greater than the base's and
// docs/upgrade/v<major>.md names the item (for example
// "workflow.step_types/timer").
//
// Usage: compat [-base ref] [-inventory path] [-upgrade-dir dir]. The base
// defaults to $BASE, else origin/main. A base with no inventory file has
// nothing to compare and passes; a base ref that does not exist fails, so a
// shallow clone cannot pass by default.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const defaultInventory = "internal/configsurface/testdata/inventory.json"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, gitShow, os.ReadFile))
}

// errNoFile is what show returns when the ref exists but has no such file.
var errNoFile = errors.New("no such file on the base")

// showFunc returns path's content at ref, errNoFile when the ref has no such
// file, or another error when the ref cannot be read.
type showFunc func(ref, path string) ([]byte, error)

// run is the command: 0 compatible or nothing to compare, 1 an item removed,
// 2 bad input.
func run(args []string, stdout, stderr io.Writer, show showFunc, readFile func(string) ([]byte, error)) int {
	fs := flag.NewFlagSet("compat", flag.ContinueOnError)
	fs.SetOutput(stderr)
	defBase := os.Getenv("BASE")
	if defBase == "" {
		defBase = "origin/main"
	}
	base := fs.String("base", defBase, "git ref whose inventory HEAD must keep")
	invPath := fs.String("inventory", defaultInventory, "HEAD's inventory file")
	upgradeDir := fs.String("upgrade-dir", "docs/upgrade", "directory of the v<major>.md upgrade notes")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	baseRaw, err := show(*base, *invPath)
	if errors.Is(err, errNoFile) {
		fmt.Fprintf(stdout, "compat: no %s on %s: nothing to compare\n", *invPath, *base)
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, "compat:", err)
		return 2
	}
	headRaw, err := readFile(*invPath)
	if err != nil {
		fmt.Fprintln(stderr, "compat:", err)
		return 2
	}
	baseInv, err := parse(baseRaw)
	if err != nil {
		fmt.Fprintf(stderr, "compat: %s on %s: %v\n", *invPath, *base, err)
		return 2
	}
	headInv, err := parse(headRaw)
	if err != nil {
		fmt.Fprintf(stderr, "compat: %s: %v\n", *invPath, err)
		return 2
	}
	note := func(major int) (string, bool) {
		b, err := readFile(filepath.Join(*upgradeDir, fmt.Sprintf("v%d.md", major)))
		if err != nil {
			return "", false
		}
		return string(b), true
	}
	problems := compare(baseInv, headInv, note)
	for _, p := range problems {
		fmt.Fprintln(stderr, "compat:", p)
	}
	if len(problems) > 0 {
		return 1
	}
	fmt.Fprintf(stdout, "compat: ok: every item on %s is still listed\n", *base)
	return 0
}

// inventory is the part of inventory.json the baseline compares: the major
// and the flattened item names.
type inventory struct {
	major int
	items []string
}

// parse flattens inventory.json the way the suite names items:
// "<section>.<list>/<name>" for a list, "<section>.<list>/<key>.<name>" for a
// map of lists (step_fields), and "<section>/<name>" for a top-level list
// (script). "deprecated" and "reserved" are metadata, not surface.
func parse(b []byte) (inventory, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		return inventory{}, err
	}
	var inv inventory
	if err := json.Unmarshal(doc["major"], &inv.major); err != nil || inv.major < 1 {
		return inventory{}, errors.New(`"major" must be a positive integer`)
	}
	for section, raw := range doc {
		switch section {
		case "major", "deprecated", "reserved":
			continue
		}
		var list []string
		if json.Unmarshal(raw, &list) == nil {
			for _, n := range list {
				inv.items = append(inv.items, section+"/"+n)
			}
			continue
		}
		var lists map[string]json.RawMessage
		if err := json.Unmarshal(raw, &lists); err != nil {
			return inventory{}, fmt.Errorf("%q is neither a list nor an object", section)
		}
		for name, raw := range lists {
			prefix := section + "." + name + "/"
			if json.Unmarshal(raw, &list) == nil {
				for _, n := range list {
					inv.items = append(inv.items, prefix+n)
				}
				continue
			}
			var byKey map[string][]string
			if err := json.Unmarshal(raw, &byKey); err != nil {
				return inventory{}, fmt.Errorf("%s.%s is neither a list nor a map of lists", section, name)
			}
			for k, ns := range byKey {
				for _, n := range ns {
					inv.items = append(inv.items, prefix+k+"."+n)
				}
			}
		}
	}
	slices.Sort(inv.items)
	return inv, nil
}

// compare returns one problem per base item missing from head, unless head's
// major is greater and head's upgrade note names the item.
func compare(base, head inventory, note func(major int) (string, bool)) []string {
	if head.major < base.major {
		return []string{fmt.Sprintf("major went down from %d to %d", base.major, head.major)}
	}
	var text string
	hasNote := false
	if head.major > base.major {
		text, hasNote = note(head.major)
	}
	var out []string
	for _, it := range base.items {
		if slices.Contains(head.items, it) {
			continue
		}
		switch {
		case head.major == base.major:
			out = append(out, fmt.Sprintf("removed: %s (listed on the base, major %d); keep it, or raise major and name it in docs/upgrade/v%d.md", it, base.major, base.major+1))
		case !hasNote:
			out = append(out, fmt.Sprintf("removed: %s (major %d -> %d, but there is no docs/upgrade/v%d.md)", it, base.major, head.major, head.major))
		case !strings.Contains(text, it):
			out = append(out, fmt.Sprintf("removed: %s (major %d -> %d, but docs/upgrade/v%d.md does not name it)", it, base.major, head.major, head.major))
		}
	}
	return out
}

// gitShow reads path at ref with git. A ref that does not resolve is an
// error; a ref without the file is errNoFile.
func gitShow(ref, path string) ([]byte, error) {
	if _, err := git("rev-parse", "-q", "--verify", ref+"^{commit}"); err != nil {
		return nil, fmt.Errorf("base %s not found; fetch it first (git fetch origin main)", ref)
	}
	spec := ref + ":" + filepath.ToSlash(path)
	if _, err := git("cat-file", "-e", spec); err != nil {
		return nil, errNoFile
	}
	return git("show", spec)
}

func git(args ...string) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // fixed git subcommands; the ref is the -base flag
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
