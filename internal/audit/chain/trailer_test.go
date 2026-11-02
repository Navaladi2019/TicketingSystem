package chain

import "testing"

func TestTrailerVerifiable(t *testing.T) {
	cases := []struct {
		name                                 string
		anchor                               bool
		count, stubCount, lastSeq, anchorSeq int64
		want                                 bool
	}{
		{"no anchor", false, 0, 0, 0, 0, false},
		{"no anchor, counts that would add up", false, 5, 0, 5, 0, false},
		{"complete", true, 5, 0, 5, 0, true},
		{"complete after an anchor", true, 3, 0, 7, 4, true},
		{"internal gap (3..7, seq 5 excluded)", true, 4, 0, 7, 2, false},
		{"tail gap (1..5, seq 5 excluded)", true, 4, 0, 5, 0, false},
		{"zero range lines over a non-empty range", true, 0, 0, 5, 0, false},
		{"last_seq = anchor_seq (empty range)", true, 0, 0, 4, 4, false},
		{"last_seq < anchor_seq", true, 0, 0, 3, 4, false},
		{"stubs fill every gap", true, 2, 3, 5, 0, true},
		{"more lines than the range", true, 6, 0, 5, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TrailerVerifiable(c.anchor, c.count, c.stubCount, c.lastSeq, c.anchorSeq); got != c.want {
				t.Errorf("TrailerVerifiable(%t, %d, %d, %d, %d) = %t, want %t",
					c.anchor, c.count, c.stubCount, c.lastSeq, c.anchorSeq, got, c.want)
			}
		})
	}
}

// lastSeq is the head read at the start, never the last line written: range
// 1..5 with seq 5 excluded writes lines 1..4. The last written seq (4) hides
// the tail gap; the head seq (5) does not.
func TestTrailerVerifiableLastSeqIsTheHead(t *testing.T) {
	const anchorSeq, written, head = 0, 4, 5
	if !TrailerVerifiable(true, written, 0, written, anchorSeq) {
		t.Fatal("precondition: the last written seq makes the gapped range look complete")
	}
	if TrailerVerifiable(true, written, 0, head, anchorSeq) {
		t.Error("range 1..5 with seq 5 excluded is verifiable with lastSeq = head")
	}
}
