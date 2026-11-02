package interp_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// Two rules on the committed histories (QA, TICK-363).

// historiesRepoDir is historiesDir relative to the repository root.
const historiesRepoDir = "internal/workflow/testdata/histories"

// --- Append-only --------------------------------------------------------

// retiredManifestRepoPath is retiredManifestPath relative to the repository
// root.
const retiredManifestRepoPath = "internal/workflow/testdata/retired-histories.json"

// TestHistoriesAppendOnly: a history on main is never changed, and never
// deleted except by a tombstone (ADR-003). A breaking interpreter change
// ships as a new GetVersion branch plus a new history, never as a re-recorded
// one, or the gate would stop proving that runs already in flight still
// replay. A history is deleted only to retire a GetVersion change's
// DefaultVersion arm (docs/quality/ci-pipeline.md, "Deploy rules (c)"): the
// deletion needs a tombstone in retired-histories.json naming its blob on
// main, the call kept with minSupported raised, and a replay failure that
// proves the retirement broke it. checkAppendOnly lists the rules (RFC-007
// rev 1 §3.3).
//
// BASE names main's commit (origin/main in the ci workflow's unit job, for
// every event, as for TestNewMigrationsLaterThanBase). The reference is main,
// what has shipped: a story branch may re-record a history main does not have
// yet. Without BASE the test skips: a laptop's origin/main may be stale.
// R-4, R-5' and R-7' need no BASE and also run as TestRetiredHistoriesManifest.
func TestHistoriesAppendOnly(t *testing.T) {
	base := os.Getenv("BASE")
	if base == "" {
		t.Skip("BASE not set: set BASE=origin/main (after git fetch origin main) to check the histories against main")
	}
	top, err := git(t, ".", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkAppendOnly(t, strings.TrimSpace(top), base, appendOnlyOpts{replay: replayAgainstWorkTree}); err != nil {
		t.Fatal(err)
	}
}

// replayRetiredFunc replays one retired history, given by name and content,
// against the code under test and returns the replay error (nil if green).
type replayRetiredFunc func(t *testing.T, name string, content []byte) error

// replayAgainstWorkTree replays content against today's nava.interpreter,
// set up as TestReplayRecordedHistories does.
func replayAgainstWorkTree(t *testing.T, name string, content []byte) error {
	t.Helper()
	h, err := parseHistory(name, content)
	if err != nil {
		return err
	}
	return replay(t, h)
}

// parseHistory decodes one protojson history, as readHistory does from disk.
func parseHistory(name string, b []byte) (history, error) {
	h, err := client.HistoryFromJSON(bytes.NewReader(b), client.HistoryJSONOptions{})
	if err != nil {
		return history{}, fmt.Errorf("%s: %w", name, err)
	}
	if len(h.GetEvents()) == 0 {
		return history{}, fmt.Errorf("%s: no events", name)
	}
	id := h.GetEvents()[0].GetWorkflowExecutionStartedEventAttributes().GetWorkflowId()
	if id == "" {
		return history{}, fmt.Errorf("%s: the first event is not a start with a workflow id", name)
	}
	return history{name: name, workflowID: id, h: h}, nil
}

// appendOnlyOpts configures checkAppendOnly. skip turns rules off by name
// ("R-1" ... "R-8"); only TestAppendOnlyCheck sets it, to show each case is
// red because of its rule.
type appendOnlyOpts struct {
	replay replayRetiredFunc
	skip   map[string]bool
}

// checkAppendOnly checks the work tree at repo against base (RFC-007 rev 1
// §3.3). Committed histories are append-only, except by a tombstone
// (ADR-003):
//
//   - R-1: a history on base whose content differs (git hash-object against
//     base's blob) fails, tombstone or not;
//   - R-2: a history on base that the work tree deleted fails unless an entry
//     has its name and its blob on base; an added entry must name a history
//     on base, with that blob;
//   - R-3: base's retired list (empty if base has no manifest) must be a
//     field-for-field prefix of the work tree's;
//   - R-4, R-5', R-7': checkRetiredManifest and scanGetVersionCalls on the
//     work tree;
//   - R-6': for every added entry, base's internal/workflow source has
//     GetVersion(ctx, change_id, workflow.DefaultVersion, ...);
//   - R-8: every added entry, read by its blob and replayed by opts.replay,
//     fails with TMPRL1100 "removed support of version -1" for its change id.
//
// An added entry is one whose history no entry on base names. It returns one
// error listing every violation.
func checkAppendOnly(t *testing.T, repo, base string, opts appendOnlyOpts) error {
	t.Helper()
	var problems []string
	add := func(rule, msg string) {
		if !opts.skip[rule] {
			problems = append(problems, rule+": "+msg)
		}
	}

	baseManifest, err := baseRetiredManifest(t, repo, base)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(retiredManifestRepoPath)))
	if err != nil {
		return fmt.Errorf("%s: %w (the manifest is never deleted; an empty one has \"retired\": [])", retiredManifestRepoPath, err)
	}
	work, err := decodeRetiredManifest(b)
	if err != nil {
		return err
	}

	// R-3: the manifest is append-only too.
	for i, be := range baseManifest.Retired {
		switch {
		case i >= len(work.Retired):
			add("R-3", fmt.Sprintf("%s retired[%d] (%s) on %s was removed; tombstones are append-only", retiredManifestRepoPath, i, be.History, base))
		case work.Retired[i] != be:
			add("R-3", fmt.Sprintf("%s retired[%d] differs from %s (%+v, was %+v); tombstones are append-only: append, never edit, reorder or remove",
				retiredManifestRepoPath, i, base, work.Retired[i], be))
		}
	}
	var added []retiredEntry
	for _, e := range work.Retired {
		if !slices.ContainsFunc(baseManifest.Retired, func(be retiredEntry) bool { return be.History == e.History }) {
			added = append(added, e)
		}
	}

	// R-1 and R-2: the histories.
	out, err := git(t, repo, "ls-tree", "-r", "--full-tree", base, "--", historiesRepoDir)
	if err != nil {
		return fmt.Errorf("git ls-tree %s -- %s: %w (fetch main first; the checkout needs its history)", base, historiesRepoDir, err)
	}
	baseBlobs := map[string]string{} // name under histories/ -> blob on base
	for line := range strings.Lines(out) {
		meta, path, ok := strings.Cut(strings.TrimRight(line, "\r\n"), "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || fields[1] != "blob" {
			continue
		}
		blob := fields[2]
		name := strings.TrimPrefix(path, historiesRepoDir+"/")
		baseBlobs[name] = blob
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(path))); errors.Is(err, os.ErrNotExist) {
			if !slices.ContainsFunc(work.Retired, func(e retiredEntry) bool { return e.History == name && e.Blob == blob }) {
				add("R-2", fmt.Sprintf("%s was deleted without a tombstone (an entry with history %q and blob %s, its blob on %s). "+
					"Restore it; or, if its deletion retires a GetVersion change's DefaultVersion arm, add a tombstone to %s "+
					"per docs/quality/ci-pipeline.md \"Deploy rules (c)\"", path, name, blob, base, retiredManifestRepoPath))
			}
			continue
		}
		got, err := git(t, repo, "hash-object", "--", path)
		if err != nil {
			return err
		}
		if strings.TrimSpace(got) != blob {
			add("R-1", path+" was changed; a history on main is never edited, tombstone or not (record a new history under a new name)")
		}
	}
	for _, e := range added {
		if baseBlobs[e.History] != e.Blob {
			add("R-2", fmt.Sprintf("tombstone %s (blob %s) does not name a history on %s with that blob; "+
				"blob is the file's id on main (git ls-tree %s -- %s/%s)", e.History, e.Blob, base, base, historiesRepoDir, e.History))
		}
	}

	// R-4, R-5', R-7' on the work tree.
	srcs, err := workflowSources(repo)
	if err != nil {
		return err
	}
	calls, err := scanGetVersionCalls(srcs)
	if err != nil {
		add("R-7'", err.Error())
	}
	ents, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(historiesRepoDir)))
	if err != nil {
		return err
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		names = append(names, e.Name())
	}
	if err := checkRetiredManifest(work, names, calls); err != nil {
		// One line per violation, each naming its rule.
		for line := range strings.Lines(err.Error()) {
			line = strings.TrimRight(line, "\r\n")
			for _, rule := range []string{"R-4", "R-5'"} {
				if strings.Contains(line, ": "+rule+": ") {
					add(rule, line)
				}
			}
		}
	}

	if len(added) > 0 {
		// R-6': base had a DefaultVersion arm to retire.
		baseCalls, err := baseGetVersionCalls(t, repo, base)
		if err != nil {
			return err
		}
		for _, e := range added {
			if !slices.ContainsFunc(baseCalls, func(c getVersionCall) bool { return c.ChangeID == e.ChangeID && c.MinSupported == -1 }) {
				add("R-6'", fmt.Sprintf("tombstone %s: %s has no workflow.GetVersion(ctx, %q, workflow.DefaultVersion, ...) under internal/workflow; "+
					"there is no DefaultVersion arm to retire", e.History, base, e.ChangeID))
			}
		}
		// R-8: the retirement is what broke each added history.
		for _, e := range added {
			content, err := git(t, repo, "cat-file", "blob", e.Blob)
			if err != nil {
				add("R-8", fmt.Sprintf("tombstone %s: %v", e.History, err))
				continue
			}
			want := []string{"TMPRL1100", "removed support of version -1", strconv.Quote(e.ChangeID)}
			rerr := opts.replay(t, e.History, []byte(content))
			switch {
			case rerr == nil:
				add("R-8", fmt.Sprintf("tombstone %s: the history still replays green; only a history that retiring %q breaks is deleted",
					e.History, e.ChangeID))
			case !containsAll(rerr.Error(), want):
				add("R-8", fmt.Sprintf("tombstone %s: replay failed with %q, want an error containing %q; it broke for another reason",
					e.History, firstLine(rerr.Error()), strings.Join(want, `", "`)))
			}
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("committed histories are append-only, except by a tombstone (ADR-003); %s is the reference:\n%s\n"+
			"A breaking interpreter change ships as a workflow.GetVersion branch plus a NEW history "+
			"(docs/quality/ci-pipeline.md, \"Changing nava.interpreter\"). A history is deleted only to retire a "+
			"DefaultVersion arm, with a tombstone (docs/quality/ci-pipeline.md, \"Deploy rules (c)\")",
			base, strings.Join(problems, "\n"))
	}
	return nil
}

// baseRetiredManifest reads base's manifest; none on base is an empty list.
func baseRetiredManifest(t *testing.T, repo, base string) (retiredManifest, error) {
	t.Helper()
	out, err := git(t, repo, "ls-tree", "--full-tree", base, "--", retiredManifestRepoPath)
	if err != nil {
		return retiredManifest{}, fmt.Errorf("git ls-tree %s -- %s: %w (fetch main first)", base, retiredManifestRepoPath, err)
	}
	if strings.TrimSpace(out) == "" {
		return retiredManifest{Format: retiredManifestFormat, Retired: []retiredEntry{}}, nil
	}
	b, err := git(t, repo, "show", base+":"+retiredManifestRepoPath)
	if err != nil {
		return retiredManifest{}, err
	}
	m, err := decodeRetiredManifest([]byte(b))
	if err != nil {
		return retiredManifest{}, fmt.Errorf("%s on %s: %w", retiredManifestRepoPath, base, err)
	}
	return m, nil
}

// baseGetVersionCalls scans base's non-test .go files under internal/workflow
// with workflowSources' rules (testdata and "."/"_" directories skipped).
func baseGetVersionCalls(t *testing.T, repo, base string) ([]getVersionCall, error) {
	t.Helper()
	const root = "internal/workflow"
	out, err := git(t, repo, "ls-tree", "-r", "--name-only", "--full-tree", base, "--", root)
	if err != nil {
		return nil, err
	}
	srcs := map[string][]byte{}
	for line := range strings.Lines(out) {
		p := strings.TrimRight(line, "\r\n")
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			continue
		}
		dirs := strings.Split(strings.TrimPrefix(p, root+"/"), "/")
		if slices.ContainsFunc(dirs[:len(dirs)-1], func(d string) bool {
			return d == "testdata" || strings.HasPrefix(d, ".") || strings.HasPrefix(d, "_")
		}) {
			continue
		}
		b, err := git(t, repo, "show", base+":"+p)
		if err != nil {
			return nil, err
		}
		srcs[p] = []byte(b)
	}
	calls, err := scanGetVersionCalls(srcs)
	if err != nil {
		return nil, fmt.Errorf("on %s: %w", base, err)
	}
	return calls, nil
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func firstLine(s string) string {
	l, _, _ := strings.Cut(s, "\n")
	return l
}

func git(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

// --- Proof the append-only check works -----------------------------------

// retiredArm is the Technical Architect's spike change (RFC-007 §3.0): the
// timer duration doubles, the old arm is gone. With minSupported set, the
// GetVersion call is kept with that minSupported (the retirement rule (c)
// requires); unset, the call was deleted.
type retiredArm struct {
	interceptor.WorkerInterceptorBase
	changeID     string
	minSupported workflow.Version // 0: the call was deleted
}

func (r *retiredArm) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &retiredArmInbound{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, arm: r}
}

type retiredArmInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	arm *retiredArm
}

func (i *retiredArmInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(&retiredArmOutbound{WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: outbound}, arm: i.arm})
}

type retiredArmOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	arm *retiredArm
}

func (o *retiredArmOutbound) NewTimer(ctx workflow.Context, d time.Duration) workflow.Future {
	if o.arm.minSupported > 0 {
		_ = workflow.GetVersion(ctx, o.arm.changeID, o.arm.minSupported, o.arm.minSupported)
	}
	return o.Next.NewTimer(ctx, 2*d)
}

// replayWith replays the retired history against nava.interpreter with
// extra in front of it.
func replayWith(extra ...interceptor.WorkerInterceptor) replayRetiredFunc {
	return func(t *testing.T, name string, content []byte) error {
		t.Helper()
		h, err := parseHistory(name, content)
		if err != nil {
			return err
		}
		return replay(t, h, extra...)
	}
}

// retiredBy: the change id's DefaultVersion arm retired, the call kept with
// minSupported 1.
func retiredBy(changeID string) replayRetiredFunc {
	return replayWith(&retiredArm{changeID: changeID, minSupported: 1})
}

func tombstoneJSON(t *testing.T, history, blob, changeID, issue string) string {
	t.Helper()
	b, err := json.Marshal(retiredEntry{History: history, Blob: blob, ChangeID: changeID, Issue: issue})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// aliasSrc is goSrc with the workflow package imported as tw.
func aliasSrc(body string) []byte {
	return []byte("package p\n\nimport tw \"go.temporal.io/sdk/workflow\"\n\nfunc f(ctx tw.Context) {\n" + body + "\n}\n")
}

// appendScratch is a scratch repository whose commit tagged "base" plays
// main.
type appendScratch struct {
	t    *testing.T
	repo string
	// blobs: name under histories/ -> its blob on base.
	blobs map[string]string
}

func (s *appendScratch) write(rel string, body []byte) {
	s.t.Helper()
	p := filepath.Join(s.repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(p, body, 0o600); err != nil {
		s.t.Fatal(err)
	}
}

func (s *appendScratch) remove(rel string) {
	s.t.Helper()
	if err := os.Remove(filepath.Join(s.repo, filepath.FromSlash(rel))); err != nil {
		s.t.Fatal(err)
	}
}

func (s *appendScratch) git(args ...string) string {
	s.t.Helper()
	out, err := git(s.t, s.repo, args...)
	if err != nil {
		s.t.Fatal(err)
	}
	return out
}

const (
	scratchWF       = "internal/workflow/interp/wf.go"
	scratchBad      = "internal/workflow/interp/bad.go"
	callOlder       = `_ = workflow.GetVersion(ctx, "older", 1, 1)`
	callStepDefault = `_ = workflow.GetVersion(ctx, "step-x", workflow.DefaultVersion, 1)`
	callStepRaised  = `_ = workflow.GetVersion(ctx, "step-x", 1, 1)`
)

func TestAppendOnlyCheck(t *testing.T) {
	t.Parallel()
	signal, err := os.ReadFile(filepath.Join(historiesDir, "signal.json"))
	if err != nil {
		t.Fatal(err)
	}
	timeout, err := os.ReadFile(filepath.Join(historiesDir, "timeout.json"))
	if err != nil {
		t.Fatal(err)
	}
	// olderTombstone is on base already: a retirement that shipped before.
	olderTombstone := tombstoneJSON(t, "gone.json", okBlob, "older", "TICK-1")

	// newScratch commits base: old.json (signal.json, the history to retire),
	// keep.json (timeout.json), a manifest holding olderTombstone (unless
	// noManifest), and wf.go with step-x still on DefaultVersion.
	newScratch := func(t *testing.T, noManifest bool, baseWF []byte) *appendScratch {
		t.Helper()
		s := &appendScratch{t: t, repo: t.TempDir()}
		s.git("init", "-q")
		s.git("config", "core.autocrlf", "false")
		s.write(historiesRepoDir+"/old.json", signal)
		s.write(historiesRepoDir+"/keep.json", timeout)
		if !noManifest {
			s.write(retiredManifestRepoPath, []byte(manifestJSON(olderTombstone)+"\n"))
		}
		if baseWF == nil {
			baseWF = goSrc(wfImport, callOlder+"\n"+callStepDefault)
		}
		s.write(scratchWF, baseWF)
		s.git("add", ".")
		s.git("-c", "user.name=replay-gate", "-c", "user.email=replay-gate@example.invalid", "-c", "commit.gpgsign=false",
			"commit", "-q", "-m", "base")
		s.git("tag", "base")
		s.blobs = map[string]string{}
		for _, n := range []string{"old.json", "keep.json"} {
			s.blobs[n] = strings.TrimSpace(s.git("rev-parse", "base:"+historiesRepoDir+"/"+n))
		}
		return s
	}
	// manifest writes the work tree's manifest: olderTombstone, then extra.
	manifest := func(s *appendScratch, extra ...string) {
		s.write(retiredManifestRepoPath, []byte(manifestJSON(append([]string{olderTombstone}, extra...)...)+"\n"))
	}
	// retire is the correct retirement of step-x: old.json deleted, its
	// tombstone appended, the call kept with minSupported raised.
	retire := func(s *appendScratch) {
		s.remove(historiesRepoDir + "/old.json")
		manifest(s, tombstoneJSON(s.t, "old.json", s.blobs["old.json"], "step-x", "TICK-397"))
		s.write(scratchWF, goSrc(wfImport, callOlder+"\n"+callStepRaised))
	}

	cases := []struct {
		name       string
		noManifest bool   // base has no manifest
		baseWF     []byte // base's wf.go; nil: older raised, step-x on DefaultVersion
		mutate     func(s *appendScratch)
		replay     replayRetiredFunc // nil: retiredBy("step-x")
		rules      []string          // nil: passes
		want       []string          // substrings of the error
	}{
		{name: "unchanged", mutate: func(*appendScratch) {}},
		{name: "a new history", mutate: func(s *appendScratch) { s.write(historiesRepoDir+"/new.json", []byte(`{"events":[3]}`)) }},
		{name: "deletion with a matching tombstone, minSupported raised, call kept", mutate: retire},
		{name: "a manifest missing on base counts as empty", noManifest: true, mutate: func(s *appendScratch) {
			retire(s)
			s.write(retiredManifestRepoPath, []byte(manifestJSON(tombstoneJSON(s.t, "old.json", s.blobs["old.json"], "step-x", "TICK-397"))))
		}},
		{name: "aliased import is matched", mutate: func(s *appendScratch) {
			retire(s)
			s.write(scratchWF, aliasSrc(`_ = tw.GetVersion(ctx, "older", 1, 1)`+"\n"+`_ = tw.GetVersion(ctx, "step-x", 1, 1)`))
		}},
		{name: "aliased import on base is matched (R-6')",
			baseWF: aliasSrc(`_ = tw.GetVersion(ctx, "older", 1, 1)` + "\n" + `_ = tw.GetVersion(ctx, "step-x", tw.DefaultVersion, 1)`),
			mutate: retire},

		{name: "R-2: deleted without a tombstone",
			mutate: func(s *appendScratch) { s.remove(historiesRepoDir + "/old.json") },
			rules:  []string{"R-2"},
			want:   []string{historiesRepoDir + "/old.json was deleted without a tombstone", `docs/quality/ci-pipeline.md "Deploy rules (c)"`}},
		{name: "R-2: tombstone with the wrong blob",
			mutate: func(s *appendScratch) {
				retire(s)
				// keep.json's blob: a real history, so only R-2 can see the mistake.
				manifest(s, tombstoneJSON(s.t, "old.json", s.blobs["keep.json"], "step-x", "TICK-397"))
			},
			rules: []string{"R-2"},
			want:  []string{"old.json was deleted without a tombstone", "does not name a history on base with that blob"}},
		{name: "R-3: base tombstone edited",
			mutate: func(s *appendScratch) {
				retire(s)
				s.write(retiredManifestRepoPath, []byte(manifestJSON(
					tombstoneJSON(s.t, "gone.json", okBlob, "older", "TICK-2"),
					tombstoneJSON(s.t, "old.json", s.blobs["old.json"], "step-x", "TICK-397"))))
			},
			rules: []string{"R-3"},
			want:  []string{"retired[0] differs from base"}},
		{name: "R-3: base tombstone removed",
			mutate: func(s *appendScratch) {
				retire(s)
				s.write(retiredManifestRepoPath, []byte(manifestJSON(tombstoneJSON(s.t, "old.json", s.blobs["old.json"], "step-x", "TICK-397"))))
			},
			rules: []string{"R-3"},
			want:  []string{"retired[0] differs from base"}},
		{name: "R-3: base tombstone removed, nothing appended",
			mutate: func(s *appendScratch) { s.write(retiredManifestRepoPath, []byte(manifestJSON())) },
			rules:  []string{"R-3"},
			want:   []string{"retired[0] (gone.json) on base was removed"}},
		{name: "R-4: retired name re-added",
			mutate: func(s *appendScratch) { retire(s); s.write(historiesRepoDir+"/gone.json", []byte(`{"events":[9]}`)) },
			rules:  []string{"R-4"},
			want:   []string{"R-4: histories/ holds a file with this retired name"}},
		{name: "R-5': call deleted",
			mutate: func(s *appendScratch) { retire(s); s.write(scratchWF, goSrc(wfImport, callOlder)) },
			rules:  []string{"R-5'"},
			want:   []string{`change_id "step-x"): R-5': no workflow.GetVersion call has this change id`}},
		{name: "R-5': minSupported still DefaultVersion",
			mutate: func(s *appendScratch) { retire(s); s.write(scratchWF, goSrc(wfImport, callOlder+"\n"+callStepDefault)) },
			rules:  []string{"R-5'"},
			want:   []string{"R-5': " + scratchWF + ":7 still supports workflow.DefaultVersion"}},
		{name: "R-5': aliased import, minSupported still DefaultVersion",
			mutate: func(s *appendScratch) {
				retire(s)
				s.write(scratchWF, aliasSrc(`_ = tw.GetVersion(ctx, "older", 1, 1)`+"\n"+`_ = tw.GetVersion(ctx, "step-x", tw.DefaultVersion, 1)`))
			},
			rules: []string{"R-5'"},
			want:  []string{"R-5': " + scratchWF + ":7 still supports workflow.DefaultVersion"}},
		{name: "R-6': change id never existed on base",
			mutate: func(s *appendScratch) {
				s.remove(historiesRepoDir + "/old.json")
				manifest(s, tombstoneJSON(s.t, "old.json", s.blobs["old.json"], "never", "TICK-397"))
				s.write(scratchWF, goSrc(wfImport, callOlder+"\n"+callStepDefault+"\n"+`_ = workflow.GetVersion(ctx, "never", 1, 1)`))
			},
			replay: retiredBy("never"),
			rules:  []string{"R-6'"},
			want:   []string{`base has no workflow.GetVersion(ctx, "never", workflow.DefaultVersion, ...)`}},
		{name: "R-6': change id never had a DefaultVersion arm on base",
			mutate: func(s *appendScratch) {
				s.remove(historiesRepoDir + "/old.json")
				manifest(s, tombstoneJSON(s.t, "old.json", s.blobs["old.json"], "older", "TICK-397"))
			},
			replay: retiredBy("older"),
			rules:  []string{"R-6'"},
			want:   []string{`base has no workflow.GetVersion(ctx, "older", workflow.DefaultVersion, ...)`}},
		{name: "R-7': non-literal change id",
			mutate: func(s *appendScratch) {
				retire(s)
				s.write(scratchBad, goSrc(wfImport, "id := \"z\"\n_ = workflow.GetVersion(ctx, id, 1, 2)"))
			},
			rules: []string{"R-7'"},
			want:  []string{scratchBad + ":7: GetVersion changeID is not a string literal"}},
		{name: "R-7': non-literal version",
			mutate: func(s *appendScratch) {
				retire(s)
				s.write(scratchBad, goSrc(wfImport, "v := workflow.Version(1)\n_ = workflow.GetVersion(ctx, \"z\", v, 2)"))
			},
			rules: []string{"R-7'"},
			want:  []string{scratchBad + ":7: GetVersion minSupported is not"}},
		{name: "R-7': dot-import",
			mutate: func(s *appendScratch) {
				retire(s)
				s.write(scratchBad, []byte("package p\n\nimport . \"go.temporal.io/sdk/workflow\"\n\nfunc g(ctx Context) {\n_ = GetVersion(ctx, \"z\", 1, 2)\n}\n"))
			},
			rules: []string{"R-7'"},
			want:  []string{scratchBad + ":3: dot-imports go.temporal.io/sdk/workflow"}},
		{name: "R-8: retired history still replays green (the call was deleted)",
			mutate: retire,
			replay: replayWith(&retiredArm{changeID: "step-x"}),
			rules:  []string{"R-8"},
			want:   []string{"tombstone old.json: the history still replays green"}},
		{name: "R-8: fails with TMPRL1100 too old to support",
			mutate: retire,
			// A history recorded before a change has no marker, so it cannot
			// produce this through the replayer; the text is the SDK's
			// (internal_event_handlers.go validateVersion, v1.49.0).
			replay: func(*testing.T, string, []byte) error {
				return errors.New(`[TMPRL1100] Workflow code is too old to support version 2 for "step-x" changeID. The maximum supported version is 1`)
			},
			rules: []string{"R-8"},
			want:  []string{"tombstone old.json: replay failed with", "too old to support", "it broke for another reason"}},
		{name: "R-8: fails with non-determinism",
			mutate: retire,
			replay: replayWith(&extraActivityBeforeTimer{}),
			rules:  []string{"R-8"},
			want:   []string{"tombstone old.json: replay failed with", "it broke for another reason"}},
		{name: "R-8: fails with TMPRL1100 for another change id",
			mutate: retire,
			replay: retiredBy("other"),
			rules:  []string{"R-8"},
			want:   []string{`removed support of version -1. for \"other\"`, "it broke for another reason"}},
		{name: "R-1: edited history",
			mutate: func(s *appendScratch) { s.write(historiesRepoDir+"/keep.json", append(slices.Clone(timeout), '\n')) },
			rules:  []string{"R-1"},
			want:   []string{historiesRepoDir + "/keep.json was changed"}},
		{name: "R-1: edit to a history that has a tombstone",
			mutate: func(s *appendScratch) {
				retire(s)
				s.write(historiesRepoDir+"/old.json", append(slices.Clone(signal), '\n'))
			},
			// R-4 sees the name too; R-1 alone must still catch the edit.
			rules: []string{"R-1", "R-4"},
			want:  []string{historiesRepoDir + "/old.json was changed", "R-4: histories/ holds a file with this retired name"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel() // each case is its own scratch repository
			s := newScratch(t, c.noManifest, c.baseWF)
			c.mutate(s)
			replayFn := c.replay
			if replayFn == nil {
				replayFn = retiredBy("step-x")
			}
			check := func(skip ...string) error {
				opts := appendOnlyOpts{replay: replayFn, skip: map[string]bool{}}
				for _, r := range skip {
					opts.skip[r] = true
				}
				return checkAppendOnly(t, s.repo, "base", opts)
			}
			err := check()
			if c.rules == nil {
				if err != nil {
					t.Fatalf("want pass, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("passed; want %v to fail", c.rules)
			}
			for _, w := range append(slices.Clone(c.want), "\n"+c.rules[0]+": ") {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
			// Red without its rule: with the case's rules off the check passes,
			// so the case fails only because of them; with one of several left
			// on, that one alone still catches it.
			if err := check(c.rules...); err != nil {
				t.Errorf("with %v skipped the check still fails, so the case does not isolate them: %v", c.rules, err)
			}
			for _, keep := range c.rules {
				if len(c.rules) == 1 {
					break
				}
				var others []string
				for _, r := range c.rules {
					if r != keep {
						others = append(others, r)
					}
				}
				if err := check(others...); err == nil || !strings.Contains(err.Error(), keep+": ") {
					t.Errorf("%s alone did not catch it: %v", keep, err)
				}
			}
		})
	}
}

// --- R1 walk ----------------------------------------------------------------

// TestHistoriesHoldReferencesOnly walks every payload of every committed
// history (ADR-001 R1, CI-7) and fails on any that is not:
//   - a start payload: the relay's outbox.Delivery whose payload is exactly
//     {definition_version_id, record_id}, or the continue-as-new
//     workerguard.RunInput carrying it with UUID-only refs;
//   - a signal delivery: an outbox.Delivery whose payload is exactly {signal};
//   - a workerguard.Directive;
//   - a UUID-only object (StepRef): every value a UUID or a list of UUIDs;
//   - an empty (binary/null) result.
//
// The one header the platform sets, nava-tenant, must be a UUID string; any
// other header fails too.
func TestHistoriesHoldReferencesOnly(t *testing.T) {
	for _, h := range loadHistories(t) {
		if errs := walkR1(h.h); len(errs) > 0 {
			t.Errorf("%s holds data that is not references only:\n%s", h.name, strings.Join(errs, "\n"))
		}
	}
}

// TestR1WalkCatchesRecordData plants a record field in a recorded history,
// in each kind of payload, and expects the walk to report it.
func TestR1WalkCatchesRecordData(t *testing.T) {
	plants := []struct {
		name string
		// pick returns the payload to overwrite in e, or nil.
		pick func(e *historypb.HistoryEvent) *commonpb.Payload
		data func(orig map[string]any) map[string]any
	}{
		{
			name: "directive result",
			pick: func(e *historypb.HistoryEvent) *commonpb.Payload {
				return first(e.GetActivityTaskCompletedEventAttributes().GetResult())
			},
			data: func(m map[string]any) map[string]any { m["priority"] = "high"; return m },
		},
		{
			name: "activity argument",
			pick: func(e *historypb.HistoryEvent) *commonpb.Payload {
				return first(e.GetActivityTaskScheduledEventAttributes().GetInput())
			},
			data: func(m map[string]any) map[string]any { m["title"] = "printer on fire"; return m },
		},
		{
			name: "start payload",
			pick: func(e *historypb.HistoryEvent) *commonpb.Payload {
				return first(e.GetWorkflowExecutionStartedEventAttributes().GetInput())
			},
			data: func(m map[string]any) map[string]any {
				m["payload"].(map[string]any)["subject"] = "printer on fire"
				return m
			},
		},
		{
			name: "signal delivery",
			pick: func(e *historypb.HistoryEvent) *commonpb.Payload {
				return first(e.GetWorkflowExecutionSignaledEventAttributes().GetInput())
			},
			data: func(m map[string]any) map[string]any {
				m["payload"].(map[string]any)["comment"] = "approved by the manager"
				return m
			},
		},
	}
	clean := readHistory(t, filepath.Join(historiesDir, "signal.json"))
	if errs := walkR1(clean.h); len(errs) > 0 {
		t.Fatalf("signal.json is not clean: %v", errs)
	}
	for _, p := range plants {
		t.Run(p.name, func(t *testing.T) {
			h := proto.Clone(clean.h).(*historypb.History)
			planted := false
			for _, e := range h.GetEvents() {
				pl := p.pick(e)
				if pl == nil {
					continue
				}
				var m map[string]any
				if err := json.Unmarshal(pl.GetData(), &m); err != nil {
					t.Fatal(err)
				}
				b, err := json.Marshal(p.data(m))
				if err != nil {
					t.Fatal(err)
				}
				pl.Data = b
				planted = true
				break
			}
			if !planted {
				t.Fatal("signal.json has no such payload to plant into")
			}
			if errs := walkR1(h); len(errs) == 0 {
				t.Fatal("the walk accepted a payload holding a record field")
			}
		})
	}
	t.Run("foreign header", func(t *testing.T) {
		h := proto.Clone(clean.h).(*historypb.History)
		start := h.GetEvents()[0].GetWorkflowExecutionStartedEventAttributes()
		p, err := converter.GetDefaultDataConverter().ToPayload("printer on fire")
		if err != nil {
			t.Fatal(err)
		}
		start.Header.Fields["subject"] = p
		if errs := walkR1(h); len(errs) == 0 {
			t.Fatal("the walk accepted a header other than nava-tenant")
		}
	})
}

func first(p *commonpb.Payloads) *commonpb.Payload {
	if len(p.GetPayloads()) == 0 {
		return nil
	}
	return p.GetPayloads()[0]
}

//nolint:gochecknoglobals // descriptors the walk matches on
var (
	payloadName = (&commonpb.Payload{}).ProtoReflect().Descriptor().FullName()
	headerName  = (&commonpb.Header{}).ProtoReflect().Descriptor().FullName()
)

// walkR1 returns one line per payload of h that is not references only. The
// lines name the event and the field path, never the payload's content.
func walkR1(h *historypb.History) []string {
	var errs []string
	for _, e := range h.GetEvents() {
		walkMessage(e.ProtoReflect(), "", func(path string, why string) {
			errs = append(errs, fmt.Sprintf("event %d (%s) %s: %s", e.GetEventId(), e.GetEventType(), path, why))
		})
	}
	return errs
}

func walkMessage(m protoreflect.Message, path string, report func(path, why string)) {
	switch m.Descriptor().FullName() {
	case payloadName:
		if why := payloadR1(m.Interface().(*commonpb.Payload)); why != "" {
			report(path, why)
		}
		return
	case headerName:
		for k, p := range m.Interface().(*commonpb.Header).GetFields() {
			if why := headerR1(k, p); why != "" {
				report(path+".fields["+k+"]", why)
			}
		}
		return
	}
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		p := path + "." + string(fd.Name())
		switch {
		case fd.IsMap():
			if fd.MapValue().Message() != nil {
				v.Map().Range(func(k protoreflect.MapKey, mv protoreflect.Value) bool {
					walkMessage(mv.Message(), p+"["+k.String()+"]", report)
					return true
				})
			}
		case fd.IsList():
			if fd.Message() != nil {
				for i := range v.List().Len() {
					walkMessage(v.List().Get(i).Message(), fmt.Sprintf("%s[%d]", p, i), report)
				}
			}
		case fd.Message() != nil:
			walkMessage(v.Message(), p, report)
		}
		return true
	})
}

func headerR1(key string, p *commonpb.Payload) string {
	if key != workerguard.TenantHeader {
		return "a header other than " + workerguard.TenantHeader
	}
	var s string
	if string(p.GetMetadata()[converter.MetadataEncoding]) != converter.MetadataEncodingJSON || json.Unmarshal(p.GetData(), &s) != nil {
		return "the tenant header is not a JSON string"
	}
	if id, err := uuid.Parse(s); err != nil || id.String() != s {
		return "the tenant header is not a canonical UUID"
	}
	return ""
}

// payloadR1 returns why p is not one of the allowed shapes, or "".
func payloadR1(p *commonpb.Payload) string {
	switch string(p.GetMetadata()[converter.MetadataEncoding]) {
	case converter.MetadataEncodingNil:
		if len(p.GetData()) != 0 {
			return "a binary/null payload with data"
		}
		return ""
	case converter.MetadataEncodingJSON:
	default:
		return "an encoding other than json/plain"
	}
	raw := json.RawMessage(p.GetData())
	switch {
	case isDirectiveJSON(raw), isDeliveryJSON(raw), isRunInputJSON(raw), isUUIDObject(raw):
		return ""
	}
	return "not a start payload, a signal delivery, a Directive or a UUID-only object"
}

// strict decodes raw into v, refusing unknown fields and trailing data.
func strict(raw json.RawMessage, v any) bool {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	return dec.Decode(v) == nil && !dec.More()
}

func isDirectiveJSON(raw json.RawMessage) bool {
	var d workerguard.Directive
	if !strict(raw, &d) || d.Step == uuid.Nil {
		return false
	}
	switch d.Kind {
	case workerguard.KindTimer, workerguard.KindWaitSignal, workerguard.KindUpdateRecord, workerguard.KindEnd:
		return true
	}
	return false
}

// deliveryPayloadOK: the relay's two payloads, exactly.
func deliveryPayloadOK(raw json.RawMessage) bool {
	var start struct {
		DefinitionVersionID uuid.UUID `json:"definition_version_id"`
		RecordID            uuid.UUID `json:"record_id"`
	}
	if strict(raw, &start) && start.DefinitionVersionID != uuid.Nil && start.RecordID != uuid.Nil {
		return true
	}
	var sig struct {
		Signal uuid.UUID `json:"signal"`
	}
	return strict(raw, &sig) && sig.Signal != uuid.Nil
}

func isDeliveryJSON(raw json.RawMessage) bool {
	var d outbox.Delivery
	return strict(raw, &d) && d.OutboxID != uuid.Nil && deliveryPayloadOK(d.Payload)
}

func isRunInputJSON(raw json.RawMessage) bool {
	var in workerguard.RunInput
	if !strict(raw, &in) || in.OutboxID == uuid.Nil || !deliveryPayloadOK(in.Payload) {
		return false
	}
	return len(in.Refs) == 0 || isUUIDObject(in.Refs)
}

// isUUIDObject: a JSON object whose every value is a canonical UUID string or
// a list of them.
func isUUIDObject(raw json.RawMessage) bool {
	var m map[string]json.RawMessage
	if !strict(raw, &m) || len(m) == 0 {
		return false
	}
	isUUID := func(r json.RawMessage) bool {
		var s string
		if json.Unmarshal(r, &s) != nil {
			return false
		}
		id, err := uuid.Parse(s)
		return err == nil && id.String() == s
	}
	for _, v := range m {
		if isUUID(v) {
			continue
		}
		var list []json.RawMessage
		if json.Unmarshal(v, &list) != nil {
			return false
		}
		for _, x := range list {
			if !isUUID(x) {
				return false
			}
		}
	}
	return true
}
