// Package grants loads db/grants.yaml, the single grant manifest (ADR-001 I3,
// PLAN-001 D-P2), and applies it to the platform database.
package grants

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/yamlstrict"
)

// Manifest is db/grants.yaml.
type Manifest struct {
	PlatformDatabase         string               `yaml:"platform_database"`
	Databases                map[string]Database  `yaml:"databases"`
	Roles                    map[string]Role      `yaml:"roles"`
	InfrastructureExceptions map[string]Exception `yaml:"infrastructure_exceptions"`
}

// Database is one database db-roles creates.
type Database struct {
	Owner string `yaml:"owner"`
}

// Role is one role's attributes and grants on the platform database.
type Role struct {
	Attributes     []string                       `yaml:"attributes"`
	PgBouncerLogin bool                           `yaml:"pgbouncer_login"`
	Database       []string                       `yaml:"database"`
	Schemas        map[string][]string            `yaml:"schemas"`
	Tables         map[string]map[string][]string `yaml:"tables"`
	Functions      map[string][]string            `yaml:"functions"`
}

// Exception is an object that breaks an I3 rule on purpose. db-roles creates
// it and keeps its grants to exactly GrantedTo; the migrator leaves it alone.
type Exception struct {
	Schema          string   `yaml:"schema"`
	Function        string   `yaml:"function"`
	SecurityDefiner bool     `yaml:"security_definer"`
	GrantedTo       []string `yaml:"granted_to"`
	Reason          string   `yaml:"reason"`
}

// Attribute names accepted under attributes.
const (
	AttrLogin       = "LOGIN"
	AttrReplication = "REPLICATION"
)

var (
	identRe    = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)
	functionRe = regexp.MustCompile(`^([a-z_][a-z0-9_]*\.)?[a-z_][a-z0-9_]*\(([a-z0-9_ ,\[\]]*)\)$`)
)

// Privileges allowed at each level. Column lists are allowed only for the
// privileges PostgreSQL grants per column.
func databasePrivs() []string { return []string{"CONNECT", "TEMPORARY", "CREATE"} }
func schemaPrivs() []string   { return []string{"USAGE", "CREATE"} }
func tablePrivs() []string {
	return []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER", "USAGE"}
}
func columnPrivs() []string   { return []string{"SELECT", "INSERT", "UPDATE", "REFERENCES"} }
func functionPrivs() []string { return []string{"EXECUTE"} }

// Parse decodes and validates a manifest.
func Parse(b []byte) (*Manifest, error) {
	var m Manifest
	if err := yamlstrict.DecodeBytes(b, &m); err != nil {
		return nil, fmt.Errorf("grants manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("grants manifest: %w", err)
	}
	return &m, nil
}

// Owner is the role that owns the platform database: the migrator.
func (m *Manifest) Owner() string { return m.Databases[m.PlatformDatabase].Owner }

// RoleNames returns the role names in a stable order.
func (m *Manifest) RoleNames() []string { return sortedKeys(m.Roles) }

// DatabaseNames returns the database names in a stable order.
func (m *Manifest) DatabaseNames() []string { return sortedKeys(m.Databases) }

// PgBouncerLogins returns the roles on PgBouncer's login allow-list.
func (m *Manifest) PgBouncerLogins() []string {
	var out []string
	for _, name := range m.RoleNames() {
		if m.Roles[name].PgBouncerLogin {
			out = append(out, name)
		}
	}
	return out
}

// HasAttribute reports whether the role lists attr.
func (r Role) HasAttribute(attr string) bool { return slices.Contains(r.Attributes, attr) }

// Validate checks the manifest's internal consistency. It does not look at
// the database.
func (m *Manifest) Validate() error {
	var errs []error
	bad := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	if !identRe.MatchString(m.PlatformDatabase) {
		bad("platform_database %q is not a lower-case identifier", m.PlatformDatabase)
	} else if _, ok := m.Databases[m.PlatformDatabase]; !ok {
		bad("platform_database %q is not listed under databases", m.PlatformDatabase)
	}
	if len(m.Roles) == 0 {
		bad("no roles")
	}
	for name, db := range m.Databases {
		if !identRe.MatchString(name) {
			bad("database %q is not a lower-case identifier", name)
		}
		if _, ok := m.Roles[db.Owner]; !ok {
			bad("database %q: owner %q is not a role in this manifest", name, db.Owner)
		}
	}

	exceptionSchemas := map[string]string{}
	exceptionFuncs := map[string]string{}
	for name, ex := range m.InfrastructureExceptions {
		if !identRe.MatchString(name) {
			bad("infrastructure_exceptions: %q is not a lower-case identifier", name)
		}
		if !identRe.MatchString(ex.Schema) || ex.Schema == "public" {
			bad("infrastructure_exceptions.%s: schema %q must be a dedicated schema", name, ex.Schema)
		}
		fn, err := normalizeFunction(ex.Function)
		if err != nil {
			bad("infrastructure_exceptions.%s: %v", name, err)
		} else if !strings.HasPrefix(fn, ex.Schema+".") {
			bad("infrastructure_exceptions.%s: function %q is not in schema %q", name, ex.Function, ex.Schema)
		}
		if strings.TrimSpace(ex.Reason) == "" {
			bad("infrastructure_exceptions.%s: reason is required", name)
		}
		if len(ex.GrantedTo) == 0 {
			bad("infrastructure_exceptions.%s: granted_to is empty", name)
		}
		for _, r := range ex.GrantedTo {
			role, ok := m.Roles[r]
			if !ok {
				bad("infrastructure_exceptions.%s: granted_to %q is not a role in this manifest", name, r)
				continue
			}
			if !slices.Contains(role.Schemas[ex.Schema], "USAGE") || !slices.Contains(role.Functions[ex.Function], "EXECUTE") {
				bad("infrastructure_exceptions.%s: role %q must list USAGE on schema %s and EXECUTE on %s", name, r, ex.Schema, ex.Function)
			}
		}
		exceptionSchemas[ex.Schema] = name
		exceptionFuncs[fn] = name
	}

	owner := m.Owner()
	for name, r := range m.Roles {
		if !identRe.MatchString(name) || strings.HasPrefix(name, "pg_") {
			bad("role %q is not a lower-case identifier outside pg_", name)
		}
		for _, a := range r.Attributes {
			if a != AttrLogin && a != AttrReplication {
				bad("role %s: attribute %q is not LOGIN or REPLICATION", name, a)
			}
		}
		if hasDup(r.Attributes) {
			bad("role %s: attributes repeat", name)
		}
		if r.PgBouncerLogin {
			// The pooler is for runtime roles only: never the owner, a
			// replication role, or a role that cannot log in.
			for dbName, db := range m.Databases {
				if db.Owner == name {
					bad("role %s: owns database %s, so it may not log in through PgBouncer", name, dbName)
				}
			}
			if r.HasAttribute(AttrReplication) || !r.HasAttribute(AttrLogin) {
				bad("role %s: pgbouncer_login needs LOGIN and no REPLICATION", name)
			}
		}
		if name == owner && (len(r.Database) > 0 || len(r.Schemas) > 0 || len(r.Tables) > 0 || len(r.Functions) > 0) {
			bad("role %s: the owner holds every privilege already; list no grants", name)
		}
		checkPrivs(bad, name+": database", r.Database, databasePrivs())
		for s, privs := range r.Schemas {
			if !identRe.MatchString(s) {
				bad("role %s: schema %q is not a lower-case identifier", name, s)
			}
			if ex, ok := exceptionSchemas[s]; ok && !slices.Contains(m.InfrastructureExceptions[ex].GrantedTo, name) {
				bad("role %s: schema %s belongs to infrastructure exception %s, which is not granted to it", name, s, ex)
			}
			checkPrivs(bad, fmt.Sprintf("%s: schema %s", name, s), privs, schemaPrivs())
		}
		for t, privs := range r.Tables {
			if _, _, err := splitTable(t); err != nil {
				bad("role %s: %v", name, err)
			}
			for p, cols := range privs {
				where := fmt.Sprintf("%s: table %s %s", name, t, p)
				checkPrivs(bad, where, []string{p}, tablePrivs())
				if len(cols) > 0 && !slices.Contains(columnPrivs(), p) {
					bad("role %s: %s is table-level only; list no columns", where, p)
				}
				for _, c := range cols {
					if !identRe.MatchString(c) {
						bad("role %s: column %q is not a lower-case identifier", where, c)
					}
				}
				if hasDup(cols) {
					bad("role %s: columns repeat", where)
				}
			}
		}
		for f, privs := range r.Functions {
			fn, err := normalizeFunction(f)
			if err != nil {
				bad("role %s: %v", name, err)
				continue
			}
			if ex, ok := exceptionFuncs[fn]; ok && !slices.Contains(m.InfrastructureExceptions[ex].GrantedTo, name) {
				bad("role %s: function %s belongs to infrastructure exception %s, which is not granted to it", name, f, ex)
			}
			checkPrivs(bad, fmt.Sprintf("%s: function %s", name, f), privs, functionPrivs())
		}
	}
	return errors.Join(errs...)
}

func checkPrivs(bad func(string, ...any), where string, privs, allowed []string) {
	for _, p := range privs {
		if !slices.Contains(allowed, p) {
			bad("%s: privilege %q is not one of %v", where, p, allowed)
		}
	}
	if hasDup(privs) {
		bad("%s: privileges repeat", where)
	}
}

// splitTable splits "schema.table" or a bare "table" (schema public).
func splitTable(t string) (schema, table string, err error) {
	schema, table = "public", t
	if s, rest, ok := strings.Cut(t, "."); ok {
		schema, table = s, rest
	}
	if !identRe.MatchString(schema) || !identRe.MatchString(table) {
		return "", "", fmt.Errorf("table %q is not schema.table or table in lower-case identifiers", t)
	}
	return schema, table, nil
}

// normalizeFunction qualifies a bare function with public and checks the
// "name(argtypes)" shape. Argument types must be written as PostgreSQL
// prints them (regprocedure), e.g. integer, not int4.
func normalizeFunction(f string) (string, error) {
	if !functionRe.MatchString(f) {
		return "", fmt.Errorf("function %q is not schema.name(argtypes)", f)
	}
	if name, _, _ := strings.Cut(f, "("); !strings.Contains(name, ".") {
		return "public." + f, nil
	}
	return f, nil
}

func hasDup(s []string) bool {
	seen := map[string]bool{}
	for _, v := range s {
		if seen[v] {
			return true
		}
		seen[v] = true
	}
	return false
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
