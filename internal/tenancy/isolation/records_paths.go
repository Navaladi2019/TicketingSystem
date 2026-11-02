package isolation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// recordPaths are the record routes (P3-14, TICK-357): the entity-type
// harness (etIso), which runs the real nava-core handler over the suite's
// database as nava_app. Tenant A's admin creates the type iso_et_rec and a
// record of it; tenant B has no such type. A case deletes every row it writes
// (etIso.cleanup) before it returns.
func recordPaths() []Path {
	db := func(name string, fn func(context.Context, *etIso) error) Path {
		return Path{Name: "records: " + name, Case: "CI-8", Role: "nava_app", DB: func(ctx context.Context, cl Cluster) error {
			e, err := newETIso(ctx, cl)
			if err != nil {
				return err
			}
			defer e.stop()
			if err := e.cleanup(); err != nil { // rows a killed earlier run left
				return err
			}
			err = fn(ctx, e)
			if cerr := e.cleanup(); err == nil {
				err = cerr
			}
			return err
		}}
	}
	return []Path{
		db("tenant B reading tenant A's record gets the missing record's 404, byte for byte", recReadOtherTenants),
		db("tenant B patching tenant A's record or reading its history gets the missing record's 404, byte for byte, and changes nothing", recPatchOtherTenants),
		db("a history cursor issued to tenant A's caller is refused for tenant B on the same URL with 400 invalid_cursor", recHistoryCursorReplay),
		db("tenant B creating under tenant A's type gets the unknown type's 404, byte for byte, and writes nothing", recCreateOtherTenantsType),
		// P3-18 (TICK-366).
		db("tenant A naming tenant B's only key gets the unknown key's 404, byte for byte, on POST and GET", recOtherTenantsOnlyKey),
		db("tenant A naming tenant B's group or principal gets a random UUID's 422, with no foreign key detail", recForeignReference),
	}
}

// recOtherTenantsOnlyKey: incident is tenant B's only key (testdata/seed.sql).
// Tenant A naming it must get the 404 of a key no tenant has, so the
// response is no oracle for another tenant's types.
func recOtherTenantsOnlyKey(ctx context.Context, e *etIso) error {
	var a, b int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FILTER (WHERE tenant_id = $1), count(*) FILTER (WHERE tenant_id = $2)
		FROM entity_definitions WHERE key = 'incident'`, TenantA, TenantB).Scan(&a, &b); err != nil {
		return err
	}
	if a != 0 || b != 1 {
		return fmt.Errorf("control: incident is defined %d times in tenant A and %d in tenant B, want 0 and 1", a, b)
	}
	before, err := e.recordCount(ctx, TenantB)
	if err != nil {
		return err
	}
	const id = "0194f2c0-0000-7000-8000-0000000000ff"
	for _, c := range []struct{ method, other, unknown, body string }{
		{http.MethodPost, "/api/v1/records/incident", "/api/v1/records/iso_et_none", `{"data": {}}`},
		{http.MethodGet, "/api/v1/records/incident/" + id, "/api/v1/records/iso_et_none/" + id, ""},
	} {
		if err := e.sameNotFound(ctx, c.method, c.body, c.other, c.unknown); err != nil {
			return err
		}
	}
	after, err := e.recordCount(ctx, TenantB)
	if err != nil {
		return err
	}
	if after != before {
		return fmt.Errorf("tenant B's records went from %d to %d", before, after)
	}
	return nil
}

// recRefSchema maps a payload property onto each reference column a record
// create can name (internal/server.fkColumn).
const recRefSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,` +
	`"required":["state"],"properties":{"state":{"type":"string","enum":["new","closed"],"default":"new"},` +
	`"group":{"type":"string","format":"uuid"},"assignee":{"type":"string","format":"uuid"},"requester":{"type":"string","format":"uuid"}}}`

func recRefTypeBody(key, prefix string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Iso %s", "number_prefix": %q, "definition": {"schema": %s,
		"fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
		"states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}}`,
		key, key, prefix, recRefSchema)
}

// recForeignReference: tenant A names tenant B's group or principal in a
// reference field. The composite foreign key refuses it (D-3-2); the 422 must
// be the one a random UUID gets, so it is no oracle for tenant B's ids, and
// must not show the constraint, the table or the id.
func recForeignReference(ctx context.Context, e *etIso) error {
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerA, recRefTypeBody("iso_et_ref", "ISR"), nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant A creating iso_et_ref = %d %s, want 201", code, body)
	}
	// Positive control: tenant A's own group and principal resolve, so a 422
	// below is the reference, not the type or the route.
	own := fmt.Sprintf(`{"data": {"group": %q, "assignee": %q, "requester": %q}}`, adminGroupA, principalA1, principalA1)
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_ref", etCallerA, own, nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant A naming its own group and principal = %d %s, want 201", code, body)
	}
	const random = "00000000-0000-4000-8000-00000000dead"
	for _, c := range []struct{ prop, foreign string }{
		{"group", GroupB1}, {"assignee", PrincipalB1}, {"requester", PrincipalB1},
	} {
		var bodies [2]string
		for i, id := range []string{random, c.foreign} {
			code, body := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_ref", etCallerA, fmt.Sprintf(`{"data": {%q: %q}}`, c.prop, id), nil)
			if code != http.StatusUnprocessableEntity {
				return fmt.Errorf("%s = %s: %d %s, want 422", c.prop, id, code, body)
			}
			for _, leak := range []string{"fkey", "constraint", "violates", "foreign key", "records", "principals", "groups", id} {
				if strings.Contains(body, leak) {
					return fmt.Errorf("%s = %s: the 422 body shows %q: %s", c.prop, id, leak, body)
				}
			}
			stripped, err := withoutInstance(body)
			if err != nil {
				return fmt.Errorf("%s = %s: %w", c.prop, id, err)
			}
			bodies[i] = stripped
		}
		if bodies[0] != bodies[1] {
			return fmt.Errorf("%s: tenant B's id gets a different 422 from a random UUID's:\n%s\n%s", c.prop, bodies[1], bodies[0])
		}
	}
	var n int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
		WHERE d.key = 'iso_et_ref'`).Scan(&n); err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%d records of iso_et_ref exist, want only the control", n)
	}
	return nil
}

// withoutInstance drops a problem body's instance, the per-request id; the
// rest must match byte for byte.
func withoutInstance(body string) (string, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return "", fmt.Errorf("body is not JSON: %w: %s", err, body)
	}
	delete(m, "instance")
	out, err := json.Marshal(m)
	return string(out), err
}

// recordCount is tenant's record count, read as the superuser past RLS.
func (e *etIso) recordCount(ctx context.Context, tenant string) (int, error) {
	var n int
	err := e.super.QueryRow(ctx, `SELECT count(*) FROM records WHERE tenant_id = $1`, tenant).Scan(&n)
	return n, err
}

// recCreate makes a record of iso_et_rec in tenant A and returns its id: the
// positive control that the type, the route and the token are live.
func recCreate(ctx context.Context, e *etIso) (string, error) {
	if err := e.create(ctx, "iso_et_rec", "ISC"); err != nil {
		return "", err
	}
	code, body := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_rec", etCallerA, `{"data": {"note": "mine"}}`, nil)
	if code != http.StatusCreated {
		return "", fmt.Errorf("control: tenant A creating a record = %d %s, want 201", code, body)
	}
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil || r.ID == "" {
		return "", fmt.Errorf("control: tenant A's record has no id: %w: %s", err, body)
	}
	if code, body := e.do(ctx, http.MethodGet, "/api/v1/records/iso_et_rec/"+r.ID, etCallerA, "", nil); code != http.StatusOK {
		return "", fmt.Errorf("control: tenant A reading its own record = %d %s, want 200", code, body)
	}
	return r.ID, nil
}

func recReadOtherTenants(ctx context.Context, e *etIso) error {
	id, err := recCreate(ctx, e)
	if err != nil {
		return err
	}
	const missing = "0194f2c0-0000-7000-8000-000000000000"
	// Tenant B's seeded types do not compile (testdata/seed.sql), so tenant B's
	// admin makes a type of its own: the type tenant B "does have".
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerBAdmin, etCreateBody("iso_et_recb", "ISB", ""), nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant B creating its own type = %d %s, want 201", code, body)
	}
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_recb", etCallerB, `{"data": {"note": "b"}}`, nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant B creating its own record = %d %s, want 201", code, body)
	}
	// Tenant B names tenant A's type and record, and its own type with
	// tenant A's record id.
	for _, c := range [][2]string{
		{"/api/v1/records/iso_et_rec/" + id, "/api/v1/records/iso_et_rec/" + missing},
		{"/api/v1/records/iso_et_recb/" + id, "/api/v1/records/iso_et_recb/" + missing},
	} {
		oc, ob := e.do(ctx, http.MethodGet, c[0], etCallerB, "", nil)
		uc, ub := e.do(ctx, http.MethodGet, c[1], etCallerB, "", nil)
		switch {
		case uc != http.StatusNotFound:
			return fmt.Errorf("control: tenant B GET %s = %d %s, want 404", c[1], uc, ub)
		case oc != http.StatusNotFound:
			return fmt.Errorf("tenant B GET %s (tenant A's) = %d %s, want 404", c[0], oc, ob)
		case ob != ub:
			return fmt.Errorf("tenant B GET %s (tenant A's) 404 differs from the missing one's:\n%s\n%s", c[0], ob, ub)
		}
	}
	return nil
}

func recCreateOtherTenantsType(ctx context.Context, e *etIso) error {
	if _, err := recCreate(ctx, e); err != nil {
		return err
	}
	body := `{"data": {"note": "forged"}}`
	oc, ob := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_rec", etCallerB, body, nil)
	uc, ub := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_none", etCallerB, body, nil)
	switch {
	case uc != http.StatusNotFound:
		return fmt.Errorf("control: tenant B creating under an unknown type = %d %s, want 404", uc, ub)
	case oc != http.StatusNotFound:
		return fmt.Errorf("tenant B creating under tenant A's type = %d %s, want 404", oc, ob)
	case ob != ub:
		return fmt.Errorf("tenant B's 404 for tenant A's type differs from the unknown type's:\n%s\n%s", ob, ub)
	}
	var n int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
		WHERE d.key = 'iso_et_rec'`).Scan(&n); err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%d records of iso_et_rec exist, want only tenant A's one", n)
	}
	return nil
}

// recPatchOtherTenants (TICK-358): PATCH and history across tenants. Tenant B
// names tenant A's record with the right ETag and gets the 404 of a record
// that does not exist, not a 412 or 428; tenant A's record stays at version 1.
func recPatchOtherTenants(ctx context.Context, e *etIso) error {
	id, err := recCreate(ctx, e)
	if err != nil {
		return err
	}
	const missing = "0194f2c0-0000-7000-8000-000000000000"
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerBAdmin, etCreateBody("iso_et_rec", "ISB", ""), nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant B creating its own type = %d %s, want 201", code, body)
	}
	patch := http.Header{"Content-Type": {"application/merge-patch+json"}, "If-Match": {`W/"1"`}}
	for _, m := range []string{http.MethodPatch, http.MethodGet} {
		suffix := ""
		if m == http.MethodGet {
			suffix = "/history"
		}
		hdr, body := http.Header(nil), ""
		if m == http.MethodPatch {
			hdr, body = patch, `{"data": {"note": "forged"}}`
		}
		oc, ob := e.do(ctx, m, "/api/v1/records/iso_et_rec/"+id+suffix, etCallerB, body, hdr)
		uc, ub := e.do(ctx, m, "/api/v1/records/iso_et_rec/"+missing+suffix, etCallerB, body, hdr)
		switch {
		case uc != http.StatusNotFound:
			return fmt.Errorf("control: tenant B %s of a missing record%s = %d %s, want 404", m, suffix, uc, ub)
		case oc != http.StatusNotFound:
			return fmt.Errorf("tenant B %s of tenant A's record%s = %d %s, want 404", m, suffix, oc, ob)
		case ob != ub:
			return fmt.Errorf("tenant B %s 404 for tenant A's record%s differs from the missing one's:\n%s\n%s", m, suffix, ob, ub)
		}
	}
	var v int64
	if err := e.super.QueryRow(ctx, `SELECT r.version FROM records r WHERE r.id = $1`, id).Scan(&v); err != nil {
		return err
	}
	if v != 1 {
		return fmt.Errorf("tenant A's record is at version %d after tenant B's patch, want 1", v)
	}
	return nil
}

// recHistoryCursorReplay (TICK-358): a history cursor decodes for the caller
// it was issued to and for nobody else, on the same URL.
func recHistoryCursorReplay(ctx context.Context, e *etIso) error {
	id, err := recCreate(ctx, e)
	if err != nil {
		return err
	}
	patch := http.Header{"Content-Type": {"application/merge-patch+json"}, "If-Match": {`W/"1"`}}
	if code, body := e.do(ctx, http.MethodPatch, "/api/v1/records/iso_et_rec/"+id, etCallerA, `{"data": {"note": "again"}}`, patch); code != http.StatusOK {
		return fmt.Errorf("control: tenant A patching its record = %d %s, want 200", code, body)
	}
	first := "/api/v1/records/iso_et_rec/" + id + "/history?limit=1"
	code, body := e.do(ctx, http.MethodGet, first, etCallerA, "", nil)
	if code != http.StatusOK {
		return fmt.Errorf("control: tenant A's first history page = %d %s, want 200", code, body)
	}
	var page struct {
		Next string `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(body), &page); err != nil || page.Next == "" {
		return fmt.Errorf("control: tenant A's first history page has no next_cursor: %w: %s", err, body)
	}
	next := first + "&cursor=" + url.QueryEscape(page.Next)
	if code, body := e.do(ctx, http.MethodGet, next, etCallerA, "", nil); code != http.StatusOK {
		return fmt.Errorf("control: tenant A replaying its own cursor = %d %s, want 200", code, body)
	}
	code, body = e.do(ctx, http.MethodGet, next, etCallerB, "", nil)
	if code != http.StatusBadRequest || !strings.Contains(body, `"invalid_cursor"`) {
		return fmt.Errorf("tenant B replaying tenant A's cursor = %d %s, want 400 invalid_cursor", code, body)
	}
	return nil
}
