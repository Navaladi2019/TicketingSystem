package fieldindex

// Default quotas (ADR-006 D12).
const (
	// DefaultPerTypeLimit is the most flagged fields one entity type may have
	// when the tenant has no tenant_limits row. An operator may raise a
	// tenant's limit up to 10 (TICK-751); publish reads it through
	// tenantlimits.PerType, which this package cannot import (the metadata
	// package imports it and may not depend on a database driver).
	DefaultPerTypeLimit = 5
	// TenantLimit is the most flagged fields a tenant may have across all its
	// entity types.
	TenantLimit = 50
)
