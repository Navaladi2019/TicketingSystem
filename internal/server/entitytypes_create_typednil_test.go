package server

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
)

// A valid create body gives parseCreate a nil *problemError. Passing it to
// errors.As as an error makes a non-nil interface holding a nil pointer, and
// errors.As calls its Unwrap, which dereferences it (PR #413 at 80a5c75,
// entitytypes.go:186): every valid POST /api/v1/entity-types panics.
func TestCreateParseOKDoesNotPanicOnSecretCheck(t *testing.T) {
	body := `{"key":"incident","name":"T","number_prefix":"INC","definition":{"schema":{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"required":["title","state"],"properties":{"title":{"type":"string"},"state":{"type":"string","enum":["open","closed"],"default":"open"}}},"fixed_columns":{"state":"state"},"states":{"initial":"open","states":["open","closed"],"transitions":[{"from":"open","to":"closed"}]}}}`
	_, perr := parseCreate(json.RawMessage(body))
	t.Logf("perr == nil: %v", perr == nil)
	var secret *entitytypes.SecretInContentError
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("errors.As(perr) on a valid create panics: %v", r)
		}
	}()
	_ = errors.As(perr, &secret)
}
