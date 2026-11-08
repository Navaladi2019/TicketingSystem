package audit

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// MaxDiffBytes is the most diff bytes one entry stores (ADR-004 R1,
// ADR-004-E1 item 2). audit_log_diff_size_check enforces the same bound.
const MaxDiffBytes = 65536

// MaxErasedIDs is the most entry ids one audit/erase entry lists
// (ADR-004-E1, R23 as amended). 1,500 ids keep the diff under MaxDiffBytes,
// so an erase entry is never truncated.
const MaxErasedIDs = 1500

// truncateDiff returns the bytes to store for diff and whether they are cut.
// At or under MaxDiffBytes, diff is stored unchanged. Over it, the stored
// bytes are "[" + raw₁ + "," + … + rawₖ + "]": the patch's first k ops, each
// exactly as it appeared in diff (never re-marshalled), with k the largest
// count that fits; "[]" when not even the first op fits (ADR-004-E1 §1b).
//
// diff must already have passed patchwalk (secrets.Check), so it is a JSON
// array of op objects.
func truncateDiff(diff []byte) (stored []byte, truncated bool, err error) {
	if len(diff) <= MaxDiffBytes {
		return diff, false, nil
	}
	var ops []json.RawMessage
	if err := json.Unmarshal(diff, &ops); err != nil {
		return nil, false, fmt.Errorf("%w: diff: not a JSON array", ErrInvalidEntry)
	}
	var b bytes.Buffer
	b.WriteByte('[')
	k := 0
	for _, op := range ops {
		extra := len(op) + 1 // the op and the closing "]"
		if k > 0 {
			extra++ // the separating ","
		}
		if b.Len()+extra > MaxDiffBytes {
			break
		}
		if k > 0 {
			b.WriteByte(',')
		}
		b.Write(op)
		k++
	}
	b.WriteByte(']')
	// The flag says ops were dropped, exactly when k < n. An over-limit diff
	// whose ops all fit once its whitespace is gone is rejoined but complete.
	return b.Bytes(), k < len(ops), nil
}

// checkEraseDiff enforces the shape of an audit/erase diff
// (docs/api/audit-export-v1.md §4 step 7): exactly one add of /erased_ids
// whose value lists 1 to MaxErasedIDs lowercase entry ids. The verifier reads
// the ids from that op only, so any other shape would leave every erased
// entry unlisted and reported as tampered (R24).
func checkEraseDiff(diff []byte) error {
	if diff == nil {
		return invalid("diff", "an erase entry lists the erased entry ids")
	}
	var ops []struct {
		Op    string            `json:"op"`
		Path  string            `json:"path"`
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(diff, &ops); err != nil || len(ops) != 1 ||
		ops[0].Op != "add" || ops[0].Path != "/erased_ids" {
		return invalid("diff", `an erase entry is exactly one add of "/erased_ids"`)
	}
	ids := ops[0].Value
	if len(ids) == 0 || len(ids) > MaxErasedIDs {
		return invalid("diff", fmt.Sprintf("an erase entry lists 1 to %d entry ids (ADR-004-E1)", MaxErasedIDs))
	}
	for _, raw := range ids {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return invalid("diff", "an erased entry id is not a string")
		}
		if u, err := uuid.Parse(s); err != nil || u.String() != s {
			return invalid("diff", "an erased entry id is not a lowercase UUID")
		}
	}
	return nil
}
