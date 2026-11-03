package server

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
)

// A valid create body gives parseCreate a nil *problemError. The handler must
// compare it as a pointer (perr != nil), never pass it to errors.As as an error:
// that is a non-nil interface holding a nil pointer, and errors.As calls its
// Unwrap, which dereferences it (PR #413 at 80a5c75: every valid POST
// /api/v1/entity-types panicked).
func TestCreateParseOKIsNilAndSecretUnwraps(t *testing.T) {
	const tail = `"fixed_columns":{"state":"state"},"states":{"initial":"open","states":["open","closed"],"transitions":[{"from":"open","to":"closed"}]}`
	body := func(extra string) string {
		return `{"key":"incident","name":"T","number_prefix":"INC","definition":{"schema":{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"required":["title","state"],"properties":{"title":{"type":"string"},"state":{"type":"string","enum":["open","closed"],"default":"open"}` + extra + `}},` + tail + `}}`
	}
	if _, perr := parseCreate(json.RawMessage(body(""))); perr != nil {
		t.Fatalf("valid create: perr = %v, want nil", perr)
	}
	_, perr := parseCreate(json.RawMessage(body(`,"pw":{"type":"string","x-nava-secret":true,"default":"v"}`)))
	var secret *entitytypes.SecretInContentError
	if perr == nil || !errors.As(error(perr), &secret) {
		t.Fatalf("secret create: perr = %v, want a SecretInContentError through Unwrap", perr)
	}
}
