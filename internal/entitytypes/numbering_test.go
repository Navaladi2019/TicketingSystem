package entitytypes

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// compiledRow is the step that fills the compiled text of a format version.
func compiledRow(t *testing.T, template string, o numbering.Overflow) step {
	t.Helper()
	c, err := numbering.Compile(template, o)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return step{scan: func(d []any) error { *d[0].(*string) = string(b); return nil }}
}

func TestResolveLink(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	tenant, def := uuid.New(), uuid.New()
	ref := NumberFormatChange{Set: true, Ref: &FormatRef{ID: uuid.New(), Version: 1}}
	tx := func(steps ...step) *fakeTx { return &fakeTx{t: t, steps: steps} }

	// Unpinning reads nothing.
	if l, err := resolveLink(ctx, tx(), tenant, def, "INC", 1, NumberFormatChange{Set: true}); err != nil || l.id != nil || l.compiled != nil {
		t.Errorf("unpin: %+v %v", l, err)
	}
	// Keeping reads the previous version's columns, and reports a failure to.
	if _, err := resolveLink(ctx, tx(found()), tenant, def, "INC", 1, NumberFormatChange{}); err != nil {
		t.Errorf("keep: %v", err)
	}
	if _, err := resolveLink(ctx, tx(fail(errDB)), tenant, def, "INC", 1, NumberFormatChange{}); !errors.Is(err, errDB) {
		t.Errorf("keep, read fails: %v", err)
	}
	// A format version that is not there, or does not read.
	if _, err := resolveLink(ctx, tx(fail(pgx.ErrNoRows)), tenant, def, "INC", 1, ref); !errors.Is(err, ErrFormatNotFound) {
		t.Errorf("not found: %v", err)
	}
	if _, err := resolveLink(ctx, tx(fail(errDB)), tenant, def, "INC", 1, ref); !errors.Is(err, errDB) {
		t.Errorf("read fails: %v", err)
	}
	// A stored form that is not JSON, or does not match its own template.
	bad := step{scan: func(d []any) error { *d[0].(*string) = "not json"; return nil }}
	if _, err := resolveLink(ctx, tx(bad), tenant, def, "INC", 1, ref); err == nil || !strings.Contains(err.Error(), "stored compiled") {
		t.Errorf("bad json: %v", err)
	}
	tampered := step{scan: func(d []any) error {
		*d[0].(*string) = `{"Head":"{prefix}","Width":3,"Tail":"","Overflow":"grow","Epoch":-1,"CompiledSHA256":"00"}`
		return nil
	}}
	if _, err := resolveLink(ctx, tx(tampered), tenant, def, "INC", 1, ref); !errors.Is(err, numbering.ErrInvalidCompiled) {
		t.Errorf("tampered: %v", err)
	}
	// The default needs no epoch: no further reads.
	if l, err := resolveLink(ctx, tx(compiledRow(t, "{prefix}{seq:7}", numbering.Grow)), tenant, def, "INC", 1, ref); err != nil || l.compiled == nil || !strings.Contains(*l.compiled, `"Epoch":0`) {
		t.Errorf("default: %+v %v", l, err)
	}
	// A template whose head clashes with epoch 0 is an overlap naming the default.
	var ov *OverlapError
	_, err := resolveLink(ctx, tx(compiledRow(t, "{prefix}{yy}{seq:3}", numbering.Refuse), fail(pgx.ErrNoRows)), tenant, def, "INC", 1, ref)
	if !errors.As(err, &ov) || ov.Template != "{prefix}{seq:7}" {
		t.Errorf("overlap with epoch 0: %v", err)
	}
	// An epoch read that fails.
	if _, err := resolveLink(ctx, tx(compiledRow(t, "{prefix}-{seq:4}", numbering.Grow), fail(errDB)), tenant, def, "INC", 1, ref); !errors.Is(err, errDB) {
		t.Errorf("epoch read fails: %v", err)
	}
	// A known template reuses its epoch.
	reuse := step{scan: func(d []any) error { *d[0].(*int) = 4; return nil }}
	l, err := resolveLink(ctx, tx(compiledRow(t, "{prefix}-{seq:4}", numbering.Grow), reuse), tenant, def, "INC", 1, ref)
	if err != nil || l.compiled == nil || !strings.Contains(*l.compiled, `"Epoch":4`) {
		t.Errorf("reuse: %+v %v", l, err)
	}
}

func TestMapFormatFKOnlyMapsTheLinkKey(t *testing.T) {
	t.Parallel()
	if err := mapFormatFK(errDB); !errors.Is(err, errDB) {
		t.Errorf("other error: %v", err)
	}
}
