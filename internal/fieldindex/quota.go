package fieldindex

import "context"

// Default quotas (ADR-006 D12).
const (
	// DefaultPerTypeLimit is the most flagged fields one entity type may have.
	DefaultPerTypeLimit = 5
	// TenantLimit is the most flagged fields a tenant may have across all its
	// entity types.
	TenantLimit = 50
)

// Tx is the caller's open transaction (a pgx.Tx from tenancy.InTenantTx). It
// is opaque here: this package stays free of a database driver, because the
// metadata package imports it and may not (internal/metadata import_test).
// S22 will assert it to the query interface it needs.
type Tx any

// PerTypeLimit returns the most flagged fields one entity type of the tenant
// may have. It is the one place publish reads the per-type limit, so S22 can
// make it per-tenant without touching publish.
func PerTypeLimit(_ context.Context, _ Tx, _ [16]byte) (int, error) {
	return DefaultPerTypeLimit, nil
}
