package catalog

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictyaml"
)

// GlobalTables is db/global-tables.yaml: the platform-schema tables that are
// not tenant tables (PLAN-001 D-P16). Every other table is a tenant table.
type GlobalTables struct {
	Tables map[string]GlobalTable `yaml:"tables"`
}

// GlobalTable is one listed table.
type GlobalTable struct {
	Reason string `yaml:"reason"`
}

var tableRe = regexp.MustCompile(`^([a-z_][a-z0-9_]{0,62}\.)?[a-z_][a-z0-9_]{0,62}$`)

// ParseGlobalTables decodes and validates db/global-tables.yaml.
func ParseGlobalTables(b []byte) (*GlobalTables, error) {
	var g GlobalTables
	if err := strictyaml.Decode(b, &g); err != nil {
		return nil, fmt.Errorf("global tables: %w", err)
	}
	var errs []error
	for name, t := range g.Tables {
		if !tableRe.MatchString(name) {
			errs = append(errs, fmt.Errorf("global tables: %q is not schema.table or table in lower-case identifiers", name))
		}
		if strings.TrimSpace(t.Reason) == "" {
			errs = append(errs, fmt.Errorf("global tables: %s: reason is required", name))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &g, nil
}

// Names returns the listed tables schema-qualified (a bare name is in
// schema public), sorted.
func (g *GlobalTables) Names() []string {
	out := []string{}
	if g == nil {
		return out
	}
	for name := range g.Tables {
		if !strings.Contains(name, ".") {
			name = "public." + name
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
