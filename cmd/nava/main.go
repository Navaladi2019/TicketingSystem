// Command nava is the offline operator CLI. Its one command today is
//
//	nava audit verify <file>
//
// which checks a nava.audit.export/v1 JSONL audit export without a database
// or network (ADR-004 R22, PLAN-005 C4, P5-03). It imports only
// internal/audit/verify and, through it, internal/audit/chain (PLAN-005
// D-5-6); arch_test.go in internal/audit/verify enforces that.
//
// Exit codes are part of the contract; see verifyUsage.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
)

// Exit codes of `nava audit verify`. Scripts depend on them; do not renumber.
const (
	ExitOK           = 0 // every line verified (redacted diffs noted, not attested)
	ExitTampered     = 1 // a check failed: the first failing line is reported
	ExitUsage        = 2 // bad arguments, or the file cannot be read
	ExitPending      = 3 // nothing tampered, but an erasure's audit/erase entry is not sealed yet
	ExitUnverifiable = 4 // not verifiable: stubs omitted (an honest stubs=false export)
	ExitEmpty        = 5 // no sealed entry in the export window: nothing to verify
)

const usage = `nava is the offline operator CLI.

Usage:
  nava audit verify <file>    verify an audit export (nava audit verify -h)
  nava help                   show this help
`

const verifyUsage = `Usage: nava audit verify <file>

Verifies a nava.audit.export/v1 JSONL audit export offline: no database, no
network. <file> is a path, or - for standard input.

It recomputes every hash with the published canonical form
(docs/api/audit-export-v1.md): envelope_sha256 = SHA-256(JCS(envelope)),
entry_hash = SHA-256(prev_hash || envelope_sha256), diff_sha256 =
SHA-256(diff_salt || diff). It checks that the range starts at the anchor
(genesis, or the stub at first_seq - 1), that every seq follows the one
before and links to its entry_hash, that the range ends at the trailer's
head_hash, that the counts match, that every entry belongs to the header's
tenant, and that each erased entry is listed by an audit/erase entry (R24).

Exit codes:
  0  ok           every line verified. A redacted diff is reported as
                  "diff not verifiable: redacted": its line is in the chain,
                  but the diff text is not attested by this file.
  1  tampered     a check failed. The first failing line is reported with
                  its line number, seq and reason code.
  2  usage        bad arguments, or the file could not be read.
  3  pending      nothing is tampered, but an erased entry is vouched for
                  only by an audit/erase entry that is not sealed yet.
                  Export and verify again after the sealer has run.
  4  not verifiable: stubs omitted
                  the export was made with stubs=false, so the entries the
                  filter excluded are missing and the chain has gaps. The
                  lines present are checked, but the range is not proven
                  complete. Export again with stubs=true to verify.
  5  empty        no sealed entry falls in the export window. Nothing is
                  verified, and nothing is wrong.

Reason codes (exit 1): envelope_sha256_mismatch, entry_hash_mismatch,
diff_sha256_mismatch, diff_missing, diff_state_mismatch,
header_tenant_mismatch, erasure_unlisted, verifiable_flag_mismatch,
anchor_missing, anchor_invalid, anchor_mismatch, seq_duplicate,
seq_out_of_order, seq_missing, count_mismatch, head_mismatch,
line_state_mismatch, line_order, header_invalid, trailer_missing,
unknown_kind, malformed_line.

What a pass proves: every line present is what was sealed, and the range is
contiguous from the anchor to head_hash. What it does not prove:
  - that a stub was an entry the filter excluded: a full entry replaced by
    its own stub still verifies, so filter completeness is not attested;
  - anything about a file rewritten consistently as a whole, or entries cut
    off the end with head_hash rewritten, or (for a time-window export) a
    range whose first entries were cut and re-anchored. Nothing
    authenticates the file itself (ADR-004 R-A1, AR-TICK-409-01).
To catch those, compare an overlapping seq's entry_hash (the anchor's, or a
previous export's head_hash) with an export you already hold.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes one command line and returns its exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 0:
		fmt.Fprint(stderr, usage)
		return ExitUsage
	case slices.Contains([]string{"help", "-h", "-help", "--help"}, args[0]):
		fmt.Fprint(stdout, usage)
		return ExitOK
	case args[0] == "audit" && len(args) > 1 && args[1] == "verify":
		return auditVerify(args[2:], stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "nava: unknown command %q\n\n%s", strings.Join(args, " "), usage)
		return ExitUsage
	}
}

func auditVerify(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if isHelp(args) {
		fmt.Fprint(stdout, verifyUsage)
		return ExitOK
	}
	fs := flag.NewFlagSet("nava audit verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(fs.Output(), verifyUsage) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitUsage
	}
	if fs.NArg() != 1 {
		fmt.Fprint(stderr, "nava audit verify: want exactly one file\n\n"+verifyUsage)
		return ExitUsage
	}
	name := fs.Arg(0)
	in := stdin
	if name != "-" {
		f, err := os.Open(name) //nolint:gosec // G304: the operator names the file to verify
		if err != nil {
			fmt.Fprintf(stderr, "nava audit verify: %v\n", err)
			return ExitUsage
		}
		defer func() { _ = f.Close() }()
		in = f
	}
	rep, err := verify.Verify(in)
	if err != nil {
		fmt.Fprintf(stderr, "nava audit verify: reading %s: %v\n", name, err)
		return ExitUsage
	}
	return report(stdout, rep)
}

// report prints the verdict and returns its exit code.
func report(w io.Writer, r *verify.Report) int {
	notes := func() {
		for _, seq := range sortedSeqs(r) {
			switch r.Lines[seq] {
			case verify.LineDiffUnverified:
				fmt.Fprintf(w, "  seq %d: diff not verifiable: redacted\n", seq)
			case verify.LinePending:
				fmt.Fprintf(w, "  seq %d: pending: erased; its audit/erase entry is not sealed yet\n", seq)
			case verify.LineVerified, verify.LineTampered:
			}
		}
	}
	rangeText := func() string {
		if r.AnchorSeq == nil || r.LastSeq == nil {
			return ""
		}
		from := "genesis"
		if *r.AnchorSeq > 0 {
			from = fmt.Sprintf("anchor seq %d", *r.AnchorSeq)
		}
		return fmt.Sprintf("seq %d..%d from %s to head_hash %s; %d entries, %d stubs, %d unsealed",
			*r.AnchorSeq+1, *r.LastSeq, from, r.HeadHash, r.Count, r.StubCount, r.UnsealedCount)
	}
	switch r.Outcome {
	case verify.OutcomePass:
		fmt.Fprintf(w, "ok: %s\n", rangeText())
		notes()
		return ExitOK
	case verify.OutcomePending:
		fmt.Fprintf(w, "pending: %s\n", rangeText())
		notes()
		return ExitPending
	case verify.OutcomeUnverifiable:
		fmt.Fprintf(w, "not verifiable: stubs omitted (%d gaps in seq; lines present checked: %d entries, %d unsealed)\n",
			r.Gaps, r.Count, r.UnsealedCount)
		notes()
		return ExitUnverifiable
	case verify.OutcomeEmpty:
		fmt.Fprintf(w, "empty: no sealed entry in the export window (%d unsealed)\n", r.UnsealedCount)
		return ExitEmpty
	case verify.OutcomeTampered:
		f := r.Failure
		at := fmt.Sprintf("line %d", f.Line)
		if f.Seq != nil {
			at += fmt.Sprintf(" (seq %d)", *f.Seq)
		}
		fmt.Fprintf(w, "tampered: %s: %s: %s\n", at, f.Reason, f.Detail)
		return ExitTampered
	}
	fmt.Fprintf(w, "nava audit verify: unknown outcome %q\n", r.Outcome)
	return ExitTampered
}

// isHelp reports whether args is exactly one help flag. Help goes to stdout
// with exit 0; the flag package would print it to stderr.
func isHelp(args []string) bool {
	return len(args) == 1 && slices.Contains([]string{"-h", "-help", "--help"}, args[0])
}

func sortedSeqs(r *verify.Report) []int64 {
	seqs := make([]int64, 0, len(r.Lines))
	for s := range r.Lines {
		seqs = append(seqs, s)
	}
	slices.Sort(seqs)
	return seqs
}
