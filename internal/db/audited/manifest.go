// Package audited loads db/audited-tables.yaml (ADR-004 decision 2, R12;
// PLAN-005 P5-11) and generates from it every writer role's audit_log grants
// and policies. nava-db migrate merges the grants into db/grants.yaml before
// applying it, converges the policies (ApplyPolicies) and runs the CI-A3
// catalogue check (Check) in its post-run gate.
package audited

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictyaml"
)

// Manifest is db/audited-tables.yaml.
type Manifest struct {
	Audited map[string]Table     `yaml:"audited"`
	Exempt  map[string]Exemption `yaml:"exempt"`
}

// Table is one audited table.
type Table struct {
	WriterRoles     []string `yaml:"writer_roles"`
	AuditedColumns  []string `yaml:"audited_columns"`
	SubjectType     string   `yaml:"subject_type"`
	SubjectIDColumn string   `yaml:"subject_id_column"`
	Attached        bool     `yaml:"attached"`
}

// Exemption is a tenant table that is not audited, and why.
type Exemption struct {
	Reason string `yaml:"reason"`
}

// LogTable is the audit log every writer's generated grants and policies are on.
const LogTable = "audit_log"

// TriggerFunction is the commit-check constraint trigger's function
// (db/migrations/20261103100000_audit_commit_check.sql).
const TriggerFunction = "audit_commit_check"

// OnboardRole is the writer whose policies carry the pinned provisioning
// predicate instead of the canonical one (R12, the ADR-001-E3 pattern).
const OnboardRole = "nava_onboard"

// InsertColumns are the R10 columns a writer may name in an INSERT: never
// occurred_at, txid, backfilled, the seal or the erasure columns.
func InsertColumns() []string {
	return []string{
		"tenant_id", "id", "domain", "action", "subject_type", "subject_id", "entity_definition_id",
		"version_before", "version_after", "actor_principal_id", "actor_system", "source", "source_ref",
		"change_set_id", "causation_id", "causation_depth", "request_id", "content_sha256", "diff_sha256",
		"diff_truncated", "diff", "diff_salt",
	}
}

// SelectColumns are the columns a writer reads (R12): the commit check's
// probe and the R2 key's version_after.
func SelectColumns() []string {
	return []string{"tenant_id", "occurred_at", "txid", "subject_type", "subject_id", "version_after"}
}

// ProbeColumns are the columns the commit check's probe reads; a role that
// lacks SELECT on any of them cannot commit a write to an attached table.
func ProbeColumns() []string {
	return []string{"tenant_id", "occurred_at", "txid", "subject_type", "subject_id"}
}

var identRe = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// Parse decodes and validates a manifest against the grant manifest m: every
// writer role must be a role of m other than its owner.
func Parse(b []byte, m *grants.Manifest) (*Manifest, error) {
	var a Manifest
	if err := strictyaml.Decode(b, &a); err != nil {
		return nil, fmt.Errorf("audited-tables manifest: %w", err)
	}
	if err := a.Validate(m); err != nil {
		return nil, fmt.Errorf("audited-tables manifest: %w", err)
	}
	return &a, nil
}

// Validate checks the manifest's internal consistency and its roles against
// m. It does not look at the database; Check does.
func (a *Manifest) Validate(m *grants.Manifest) error {
	var errs []error
	bad := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	if len(a.Audited) == 0 {
		bad("no audited tables")
	}
	for name, t := range a.Audited {
		if _, _, err := grants.SplitTable(name); err != nil {
			bad("audited: %v", err)
		}
		if name == LogTable || name == "public."+LogTable {
			bad("audited.%s: the audit log cannot audit itself", name)
		}
		if _, ok := a.Exempt[name]; ok {
			bad("audited.%s: also listed under exempt", name)
		}
		if len(t.WriterRoles) == 0 {
			bad("audited.%s: writer_roles is empty", name)
		}
		for _, r := range t.WriterRoles {
			if _, ok := m.Roles[r]; !ok {
				bad("audited.%s: writer role %q is not a role in db/grants.yaml", name, r)
			} else if r == m.Owner() {
				bad("audited.%s: writer role %q is the owner, which needs no generated grant", name, r)
			}
		}
		if hasDup(t.WriterRoles) {
			bad("audited.%s: writer_roles repeat", name)
		}
		for _, c := range t.AuditedColumns {
			if !identRe.MatchString(c) {
				bad("audited.%s: audited column %q is not a lower-case identifier", name, c)
			}
		}
		if hasDup(t.AuditedColumns) {
			bad("audited.%s: audited_columns repeat", name)
		}
		if !audit.SubjectType(t.SubjectType).Valid() {
			bad("audited.%s: subject_type %q is not registered in internal/audit/subjects.go", name, t.SubjectType)
		}
		if !identRe.MatchString(t.SubjectIDColumn) {
			bad("audited.%s: subject_id_column %q is not a lower-case identifier", name, t.SubjectIDColumn)
		}
	}
	for name, e := range a.Exempt {
		if _, _, err := grants.SplitTable(name); err != nil {
			bad("exempt: %v", err)
		}
		if strings.TrimSpace(e.Reason) == "" {
			bad("exempt.%s: reason is required", name)
		}
	}
	return errors.Join(errs...)
}

// Writers returns every writer role of every audited table, sorted.
func (a *Manifest) Writers() []string {
	var out []string
	for _, t := range a.Audited {
		out = append(out, t.WriterRoles...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// TableNames returns the audited table names, sorted.
func (a *Manifest) TableNames() []string { return slices.Sorted(maps.Keys(a.Audited)) }

// MergeGrants returns a copy of m with every writer's generated audit_log
// grants (R12) added: INSERT on InsertColumns and SELECT on SelectColumns.
// A role whose db/grants.yaml entry already lists columns for that
// privilege keeps them too (nava_worker's sealer SELECT); a table-level
// grant stays table-level. m is not modified. Merging twice gives the same
// manifest.
func (a *Manifest) MergeGrants(m *grants.Manifest) (*grants.Manifest, error) {
	out := *m
	out.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		out.Roles[name] = r
	}
	for _, w := range a.Writers() {
		r, ok := out.Roles[w]
		if !ok {
			return nil, fmt.Errorf("audited-tables manifest: writer role %q is not a role in db/grants.yaml", w)
		}
		tables := make(map[string]map[string][]string, len(r.Tables)+1)
		for t, privs := range r.Tables {
			tables[t] = privs
		}
		privs := make(map[string][]string, len(tables[LogTable])+2)
		for p, cols := range tables[LogTable] {
			privs[p] = slices.Clone(cols)
		}
		for p, cols := range map[string][]string{"INSERT": InsertColumns(), "SELECT": SelectColumns()} {
			have, listed := privs[p]
			if listed && len(have) == 0 {
				continue // table-level already
			}
			privs[p] = union(have, cols)
		}
		tables[LogTable] = privs
		r.Tables = tables
		out.Roles[w] = r
	}
	if err := out.Validate(); err != nil {
		return nil, fmt.Errorf("grants manifest after merging the audited-tables grants: %w", err)
	}
	return &out, nil
}

// union is a followed by every element of b that a lacks.
func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, v := range b {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func hasDup(s []string) bool {
	c := slices.Clone(s)
	slices.Sort(c)
	return len(slices.Compact(c)) != len(s)
}

// Load parses the grant manifest (db/grants.yaml) and the audited-tables
// manifest (db/audited-tables.yaml) and returns the effective grant
// manifest, with every writer's generated audit_log grants merged in, and
// the audited-tables manifest. It is what nava-db migrate applies and what
// the catalogue gate compares against; db-roles needs only the roles, which
// the merge leaves alone.
func Load(grantsYAML, auditedYAML []byte) (*grants.Manifest, *Manifest, error) {
	m, err := grants.Parse(grantsYAML)
	if err != nil {
		return nil, nil, err
	}
	a, err := Parse(auditedYAML, m)
	if err != nil {
		return nil, nil, err
	}
	merged, err := a.MergeGrants(m)
	if err != nil {
		return nil, nil, err
	}
	return merged, a, nil
}
