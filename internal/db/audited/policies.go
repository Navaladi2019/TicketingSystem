package audited

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
)

// The generated audit_log policies (R12). The canonical pair covers every
// writer but nava_onboard; the onboard pair carries the pinned provisioning
// predicate (catalog.PinnedTenantPolicies). Every policy on audit_log whose
// name starts with GeneratedPrefix or OnboardPrefix is owned by
// ApplyPolicies: it creates the missing ones, rebuilds a drifted one and
// drops any other.
const (
	GeneratedPrefix = "audit_log_writer_"
	OnboardPrefix   = "audit_log_onboard_"
)

const (
	canonicalPred = "tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid"
	onboardPred   = canonicalPred + " AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning')"
)

// Policy is one generated policy: its name and the form pg_policies shows
// for it (command, sorted roles, deparsed USING and WITH CHECK).
type Policy struct {
	Name string
	catalog.Policy
	pred string // the predicate as written in CREATE POLICY
}

// Policies returns the generated audit_log policies, sorted by name. A pair
// whose role list would be empty is left out, so ApplyPolicies drops it.
func (a *Manifest) Policies() []Policy {
	var canonical []string
	onboard := false
	for _, w := range a.Writers() {
		if w == OnboardRole {
			onboard = true
			continue
		}
		canonical = append(canonical, w)
	}
	var out []Policy
	if len(canonical) > 0 {
		roles := strings.Join(canonical, ",")
		want := catalog.TenantPred("tenant_id")
		out = append(out,
			Policy{GeneratedPrefix + "insert", catalog.Policy{Cmd: "INSERT", Roles: roles, Check: want}, canonicalPred},
			Policy{GeneratedPrefix + "select", catalog.Policy{Cmd: "SELECT", Roles: roles, Qual: want}, canonicalPred})
	}
	if onboard {
		pinned := catalog.PinnedTenantPolicies()
		for _, cmd := range []string{"insert", "select"} {
			name := OnboardPrefix + cmd
			out = append(out, Policy{name, pinned["public."+LogTable+"."+name], onboardPred})
		}
	}
	slices.SortFunc(out, func(x, y Policy) int { return strings.Compare(x.Name, y.Name) })
	return out
}

func (p Policy) createSQL() string {
	clause := "USING"
	if p.Cmd == "INSERT" {
		clause = "WITH CHECK"
	}
	roles := strings.Split(p.Roles, ",")
	for i, r := range roles {
		roles[i] = pgx.Identifier{r}.Sanitize()
	}
	return fmt.Sprintf("CREATE POLICY %s ON public.%s FOR %s TO %s %s (%s)",
		pgx.Identifier{p.Name}.Sanitize(), LogTable, p.Cmd, strings.Join(roles, ", "), clause, p.pred)
}

// PolicyResult lists what one ApplyPolicies changed, by policy name.
type PolicyResult struct {
	Created []string
	Dropped []string
}

// Changed reports whether ApplyPolicies changed anything.
func (r PolicyResult) Changed() bool { return len(r.Created) > 0 || len(r.Dropped) > 0 }

// policyLockKey serialises concurrent appliers (pg_advisory_xact_lock).
const policyLockKey = "nava.audited.policies"

// ApplyPolicies makes the generated policies on audit_log equal Policies(),
// in one transaction, as the table's owner (the migrator). A drifted policy
// is dropped and created again. Running it again changes nothing.
func (a *Manifest) ApplyPolicies(ctx context.Context, conn *pgx.Conn, log *slog.Logger) (PolicyResult, error) {
	var res PolicyResult
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", policyLockKey); err != nil {
		return res, err
	}
	have, err := generatedPolicies(ctx, tx)
	if err != nil {
		return res, err
	}
	want := map[string]Policy{}
	for _, p := range a.Policies() {
		want[p.Name] = p
	}
	for _, name := range slices.Sorted(maps.Keys(have)) {
		if w, ok := want[name]; ok && w.Policy == have[name] {
			continue
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf("DROP POLICY %s ON public.%s", pgx.Identifier{name}.Sanitize(), LogTable)); err != nil {
			return res, fmt.Errorf("drop policy %s: %w", name, err)
		}
		log.InfoContext(ctx, "dropped generated policy", slog.String("policy", name))
		res.Dropped = append(res.Dropped, name)
		delete(have, name)
	}
	for _, p := range a.Policies() {
		if _, ok := have[p.Name]; ok {
			continue
		}
		if _, err := tx.Exec(ctx, p.createSQL()); err != nil {
			return res, fmt.Errorf("create policy %s: %w", p.Name, err)
		}
		log.InfoContext(ctx, "created generated policy", slog.String("policy", p.Name), slog.String("roles", p.Roles))
		res.Created = append(res.Created, p.Name)
	}
	// Read back: the deparsed form must be the one the catalogue gate pins.
	after, err := generatedPolicies(ctx, tx)
	if err != nil {
		return res, err
	}
	for name, p := range want {
		if after[name] != p.Policy {
			return res, fmt.Errorf("generated policy %s reads back as %q, want %q", name, after[name], p.Policy)
		}
	}
	return res, tx.Commit(ctx)
}

// generatedPolicies reads the policies on audit_log that ApplyPolicies owns.
func generatedPolicies(ctx context.Context, q catalog.Querier) (map[string]catalog.Policy, error) {
	rows, err := q.Query(ctx, `
		SELECT policyname::text, cmd, roles::text[], coalesce(qual, ''), coalesce(with_check, '')
		FROM pg_catalog.pg_policies
		WHERE schemaname = 'public' AND tablename = $1
		  AND (starts_with(policyname::text, $2) OR starts_with(policyname::text, $3))`,
		LogTable, GeneratedPrefix, OnboardPrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]catalog.Policy{}
	for rows.Next() {
		var name string
		var roles []string
		var p catalog.Policy
		if err := rows.Scan(&name, &p.Cmd, &roles, &p.Qual, &p.Check); err != nil {
			return nil, err
		}
		slices.Sort(roles)
		p.Roles = strings.Join(roles, ",")
		out[name] = p
	}
	return out, rows.Err()
}
