package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// maskingAuthz allows every row but applies the real mask write check
// (authz.CheckWriteFields) with a policy that masks "name" from everyone:
// nobody may read or write it.
type maskingAuthz struct{ fakeAuthz }

var nameMaskedFromEveryone = &authz.Policy{Masks: []authz.Mask{{Field: "name"}}}

func (maskingAuthz) CanWriteFields(_ context.Context, _ pgx.Tx, p identity.Principal, _ string, before, after map[string]any) error {
	return authz.CheckWriteFields(nameMaskedFromEveryone, p, before, after)
}

// Adversarial (security review of PR #526): a caller who can neither read
// nor write the masked field "name" must not learn its value from PATCH.
// Sending {"name": <guess>} must be refused with field_not_writable whatever
// the guess. Today a correct guess equals the stored value, so it is not a
// "written" field, the write goes through (200) and a wrong guess is a 422:
// a value oracle for any masked field with a small domain (enum, boolean).
func TestPatchingAMaskedFieldIsRefusedEvenWithItsStoredValue(t *testing.T) {
	f := newFixture()
	p := unitPipeline(maskingAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1})
	for _, guess := range []string{"wrong-guess", "the-secret"} {
		t.Run(guess, func(t *testing.T) {
			rec := storedRecord{tenant: f.tenant, id: uuid.New(), def: f.defID, key: "asset", version: 1, schemaVersion: 1,
				state: "active", data: `{"name":"the-secret"}`}
			tx := f.tx(t, stmt{match: sqlGet, scan: rec.scan}, stmt{match: sqlPayload, err: errBoom})
			_, err := p.Write(t.Context(), tx, f.update(rec.id, map[string]any{"name": guess}))
			var nw *authz.ErrFieldNotWritable
			if !errors.As(err, &nw) {
				t.Fatalf("PATCH name=%q: %v, want *authz.ErrFieldNotWritable for every guess (no value oracle)", guess, err)
			}
		})
	}
}
