package store

import "context"

// SystemConfigStore manages per-tenant configuration settings.
// Non-secret, plain-text key-value pairs. Use ConfigSecretsStore for secrets.
//
// Resolution: tenant-specific → MasterTenantID fallback → not found.
// All entries are stored per tenant_id. No sentinel UUID.
type SystemConfigStore interface {
	// Get returns the config value. Checks tenant-specific first, falls back to MasterTenantID.
	Get(ctx context.Context, key string) (string, error)
	// Set stores a config value for the current tenant (from context).
	// FAILS CLOSED: a context with no tenant is an error, never a silent write into
	// the master bucket. (Master-scoped callers carry MasterTenantID in the context,
	// so they resolve normally.) Implemented in PGSystemConfigStore.Set.
	Set(ctx context.Context, key, value string) error
	// Delete removes a config value for the current tenant.
	Delete(ctx context.Context, key string) error
	// List returns all configs visible to the current tenant (master merged with tenant overrides).
	List(ctx context.Context) (map[string]string, error)
}
