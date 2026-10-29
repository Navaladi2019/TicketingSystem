package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// landed against a real repository: a squash merge keeps the tree, not the
// commits, which is the case ancestry cannot see (TICK-322).
func TestLandedSquashMerge(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.CommandContext(context.Background(), "git", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(msg string) string {
		run("add", "-A")
		run("-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", msg)
		return run("rev-parse", "HEAD")
	}

	run("init", "-q", "-b", "main")
	write("a", "base\n")
	commit("base")
	run("checkout", "-q", "-b", "story")
	write("b", "task 1\n")
	storyHead := commit("task 1")

	// Squash-merge the story into main, then main moves on.
	run("checkout", "-q", "main")
	write("b", "task 1\n")
	commit("story (#1)")
	merged := time.Now().Add(-time.Minute)
	write("c", "other\n")
	commit("unrelated")

	check := func(base, head string) bool {
		t.Helper()
		ok, err := landed("origin", "main", base, head, merged)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}
	if !check(storyHead, storyHead) {
		t.Fatal("the squash-merged story head must be on main")
	}

	// A task PR merged into the story branch after the squash.
	run("checkout", "-q", "story")
	write("d", "late task\n")
	late := commit("late task")
	if check(storyHead, late) {
		t.Fatal("a change after the story PR is not on main")
	}
	if check("", late) {
		t.Fatal("orphan check: the late change is not on main")
	}

	// A sync PR brings it to main: on main again.
	run("checkout", "-q", "main")
	write("d", "late task\n")
	commit("sync late task (#2)")
	if !check(storyHead, late) {
		t.Fatal("after the sync PR the story branch is on main")
	}

	// main edits the synced file again: merging the old branch into the tip
	// now conflicts, but the change did land (TICK-322 review, item 1).
	write("d", "late task, edited on main\n")
	commit("edit d on main")
	if !check(storyHead, late) {
		t.Fatal("a later edit on main must not unland the synced change")
	}
	if !check("", late) {
		t.Fatal("orphan check: a later edit on main must not unland the synced change")
	}

	// A conflicting late change counts as not on main.
	run("checkout", "-q", "story")
	write("c", "conflicts with main\n")
	clash := commit("clash")
	if check(storyHead, clash) {
		t.Fatal("a conflicting change is not on main")
	}
}
