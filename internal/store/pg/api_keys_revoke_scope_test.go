package pg

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/store"
)

// API-key revocation is a security boundary: a tenant admin must be confined to
// its own tenant's keys, while the platform owner (who provisions customer
// tenants and mints dashboard tokens FOR them) must be able to revoke any key.
//
// The defect this pins: Revoke's WHERE clause carried `OR tenant_id IS NULL`,
// so any tenant-scoped caller could revoke a system-level key, and an owner
// targeting another tenant's key matched zero rows and surfaced as a 404.
//
// The store is the only shared chokepoint for the three transports (HTTP, WS,
// MCP). HTTP and WS each carry a role in ctx; MCP carries only a concrete
// tenant. Each context shape is reproduced here so the SQL is pinned against
// the callers that actually build it, not against a convenient one.
//
// Runs against a real Postgres (hooksTestDB, env-gated): row counts are the
// contract, and only a database can prove that a non-matching UPDATE affected
// zero rows and left the key live.
func TestAPIKeyRevokeTenantScope(t *testing.T) {
	db := hooksTestDB(t)
	s := NewPGAPIKeyStore(db)

	// tenant admin revoking its own tenant's key is the happy path and must
	// keep working.
	t.Run("tenant admin revokes its own key", func(t *testing.T) {
		tenant, _ := seedTenantAndAgent(t, db)
		ctx := store.WithTenantID(context.Background(), tenant)
		key := seedRevokeKey(t, db, s, tenant)

		if err := s.Revoke(ctx, key.ID, ""); err != nil {
			t.Fatalf("own key must revoke: %v", err)
		}
		if !apiKeyRevoked(t, db, key.ID) {
			t.Fatal("key should be revoked")
		}
	})

	// Another tenant's key is invisible: zero rows, sql.ErrNoRows, key live.
	t.Run("tenant admin cannot revoke another tenant's key", func(t *testing.T) {
		tenant, _ := seedTenantAndAgent(t, db)
		other, _ := seedTenantAndAgent(t, db)
		ctx := store.WithTenantID(context.Background(), tenant)
		key := seedRevokeKey(t, db, s, other)

		err := s.Revoke(ctx, key.ID, "")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("want sql.ErrNoRows, got %v", err)
		}
		if apiKeyRevoked(t, db, key.ID) {
			t.Fatal("another tenant's key must not be revoked")
		}
	})

	// The strict-equality case: the old `OR tenant_id IS NULL` arm let this
	// succeed. A NULL-tenant (system) key is revocable only by the owner.
	t.Run("tenant admin cannot revoke a NULL-tenant key", func(t *testing.T) {
		tenant, _ := seedTenantAndAgent(t, db)
		ctx := store.WithTenantID(context.Background(), tenant)
		key := seedRevokeKey(t, db, s, uuid.Nil)

		err := s.Revoke(ctx, key.ID, "")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("want sql.ErrNoRows, got %v", err)
		}
		if apiKeyRevoked(t, db, key.ID) {
			t.Fatal("a system-level key must not be revoked by a tenant admin")
		}
	})

	// HTTP owner: enrichContext (internal/http/auth.go) sets the owner role
	// while the caller's tenant stays concrete. The store's owner bypass must
	// win over the tenant predicate, or this is the reported 404.
	t.Run("owner (HTTP context) revokes another tenant's key", func(t *testing.T) {
		ownerTenant, _ := seedTenantAndAgent(t, db)
		target, _ := seedTenantAndAgent(t, db)
		ctx := store.WithRole(store.WithTenantID(context.Background(), ownerTenant), store.RoleOwner)
		key := seedRevokeKey(t, db, s, target)

		if err := s.Revoke(ctx, key.ID, ""); err != nil {
			t.Fatalf("owner must revoke a cross-tenant key: %v", err)
		}
		if !apiKeyRevoked(t, db, key.ID) {
			t.Fatal("key should be revoked by owner")
		}
	})

	t.Run("owner (HTTP context) revokes a NULL-tenant key", func(t *testing.T) {
		ownerTenant, _ := seedTenantAndAgent(t, db)
		ctx := store.WithRole(store.WithTenantID(context.Background(), ownerTenant), store.RoleOwner)
		key := seedRevokeKey(t, db, s, uuid.Nil)

		if err := s.Revoke(ctx, key.ID, ""); err != nil {
			t.Fatalf("owner must revoke a system-level key: %v", err)
		}
		if !apiKeyRevoked(t, db, key.ID) {
			t.Fatal("key should be revoked by owner")
		}
	})

	// WS owner: the router injects the connect role into ctx
	// (internal/gateway/router.go), so the context has the same shape as the
	// HTTP owner case. This is the transport the reported task believed was
	// missing the role; the test states the shape the router actually builds.
	t.Run("owner (WS context) revokes another tenant's key", func(t *testing.T) {
		clientTenant, _ := seedTenantAndAgent(t, db)
		target, _ := seedTenantAndAgent(t, db)
		ctx := store.WithRole(store.WithTenantID(context.Background(), clientTenant), store.RoleOwner)
		key := seedRevokeKey(t, db, s, target)

		if err := s.Revoke(ctx, key.ID, ""); err != nil {
			t.Fatalf("owner must revoke a cross-tenant key over WS: %v", err)
		}
		if !apiKeyRevoked(t, db, key.ID) {
			t.Fatal("key should be revoked by owner")
		}
	})

	// MCP: WithHTTPContextFunc sets only a concrete tenant, never a role
	// (internal/mcp/crud_server.go). It must fail closed on a NULL-tenant key
	// rather than inheriting the old `OR tenant_id IS NULL` arm.
	t.Run("MCP tenant context cannot revoke a NULL-tenant key", func(t *testing.T) {
		tenant, _ := seedTenantAndAgent(t, db)
		ctx := store.WithTenantID(context.Background(), tenant)
		key := seedRevokeKey(t, db, s, uuid.Nil)

		err := s.Revoke(ctx, key.ID, "")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("want sql.ErrNoRows, got %v", err)
		}
		if apiKeyRevoked(t, db, key.ID) {
			t.Fatal("MCP must not revoke a system-level key")
		}
	})

	// The ownerID filter is orthogonal and unchanged: a mismatched owner must
	// still match zero rows even when the tenant matches.
	t.Run("ownerID filter still confines the update", func(t *testing.T) {
		tenant, _ := seedTenantAndAgent(t, db)
		ctx := store.WithTenantID(context.Background(), tenant)
		key := seedRevokeKey(t, db, s, tenant)
		if _, err := db.Exec(`UPDATE api_keys SET owner_id = $1 WHERE id = $2`, "user-b", key.ID); err != nil {
			t.Fatalf("set owner_id: %v", err)
		}

		if err := s.Revoke(ctx, key.ID, "user-a"); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("owner mismatch must not revoke; got %v", err)
		}
		if apiKeyRevoked(t, db, key.ID) {
			t.Fatal("key must not be revoked on owner mismatch")
		}
		if err := s.Revoke(ctx, key.ID, "user-b"); err != nil {
			t.Fatalf("owner match must revoke: %v", err)
		}
		if !apiKeyRevoked(t, db, key.ID) {
			t.Fatal("key should be revoked on owner match")
		}
	})
}

// seedRevokeKey inserts an API key bound to tenantID (uuid.Nil = NULL tenant,
// i.e. a system-level key) and removes it afterwards. NULL-tenant rows are not
// caught by the tenant cleanup, so the id-based cleanup is unconditional.
func seedRevokeKey(t *testing.T, db *sql.DB, s *PGAPIKeyStore, tenantID uuid.UUID) *store.APIKeyData {
	t.Helper()
	now := time.Now()
	key := &store.APIKeyData{
		ID:        store.GenNewID(),
		Name:      "revoke-scope",
		Prefix:    "gclw_rvk",
		KeyHash:   "hash-" + uuid.New().String(),
		Scopes:    []string{"operator.read"},
		TenantID:  tenantID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Create(context.Background(), key); err != nil {
		t.Fatalf("create key: %v", err)
	}
	t.Cleanup(func() { db.Exec("DELETE FROM api_keys WHERE id = $1", key.ID) })
	return key
}

func apiKeyRevoked(t *testing.T, db *sql.DB, id uuid.UUID) bool {
	t.Helper()
	var revoked bool
	if err := db.QueryRow(`SELECT revoked FROM api_keys WHERE id = $1`, id).Scan(&revoked); err != nil {
		t.Fatalf("read revoked flag: %v", err)
	}
	return revoked
}
