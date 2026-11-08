package read

import (
	"errors"
	"testing"
)

func TestHistoryTooLongError(t *testing.T) {
	err := error(&HistoryTooLongError{Entries: 501, Max: 500})
	if !errors.Is(err, ErrHistoryTooLong) {
		t.Errorf("errors.Is(%v, ErrHistoryTooLong) = false", err)
	}
	if got, want := err.Error(), "audit/read: record history is longer than the cap: 501 entries, cap 500"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestMaxRecordEntriesDefault(t *testing.T) {
	if got := (&Reader{}).maxRecordEntries(); got != DefaultMaxRecordEntries {
		t.Errorf("default cap = %d", got)
	}
	if got := (&Reader{MaxRecordEntries: 7}).maxRecordEntries(); got != 7 {
		t.Errorf("set cap = %d", got)
	}
}
