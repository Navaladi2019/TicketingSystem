package entitytypes

import (
	"errors"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// A stored version that no longer compiles is a store defect. Its error must
// not match *metadata.ErrInvalidDefinition, or the publish handler would answer
// it as the caller's 422 with pointers into the stored schema; it must stay a
// plain error (500).
func TestCompileOfAStoredVersionIsNeverAnInvalidDefinition(t *testing.T) {
	t.Parallel()
	_, err := compile(&Version{Key: "incident", Version: 3, NumberPrefix: "INC", Schema: []byte(`{"type":"array"}`)})
	if err == nil {
		t.Fatal("compile of a broken stored version succeeded")
	}
	var ie *metadata.ErrInvalidDefinition
	if errors.As(err, &ie) {
		t.Fatalf("stored-version compile error matches *ErrInvalidDefinition: %v", err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("stored-version compile error is ErrNotFound: %v", err)
	}
}
