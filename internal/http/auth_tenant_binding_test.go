package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/crypto"
	"github.com/mnh-shop/7alosclaw/internal/permissions"
	"github.com/mnh-shop/7alosclaw/internal/store"
)

// TestResolveAuth_TenantBoundKeyCannotBeRescoped pins the tenant-binding
// invariant for API-key auth: a key bound to a tenant authenticates as THAT
// tenant, and the X-ZalosClaw-Tenant-Id header never re-scopes it.
//
// Source of truth:
//   - auth.go:177-259 resolveAuthWithBearer. The system-key branch (208-215)
//     consults X-ZalosClaw-Tenant-Id; the tenant-bound branch (216-219) sets
//     res.TenantID = keyData.TenantID with no header consultation.
//   - api_keys.go:97-111 minting. An owner may target any tenant_id; a
//     non-owner is forced to its own tenant and the body's tenant_id ignored.
//   - gateway/router.go:195-244 does the same on the WS connect path; the
//     tenant-bound branch at 233-240 likewise ignores params.TenantID.
//   - permissions/policy.go:40-57 scope vocabulary; api_key_cache.go:76-80
//     collapses scopes into one role via permissions.RoleFromScopes
//     (policy.go:155-169) at auth time.
//
// The table deliberately includes the one path that MAY re-scope (a
// system-level key, tenant_id NULL) so the asymmetry is stated, not hidden.
func TestResolveAuth_TenantBoundKeyCannotBeRescoped(t *testing.T) {
	tenantA := uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	tenantB := uuid.MustParse("00000000-0000-0000-0000-0000000000b2")

	const (
		boundToken  = "api-key-bound-to-tenant-a"
		systemToken = "api-key-system-level"
		unknownKey  = "api-key-that-was-revoked"
	)

	keys := &fakeAPIKeyStore{byHash: map[string]*store.APIKeyData{
		crypto.HashAPIKey(boundToken): {
			ID:       uuid.New(),
			TenantID: tenantA,
			Scopes:   []string{string(permissions.ScopeWrite)},
		},
		crypto.HashAPIKey(systemToken): {
			ID:       uuid.New(),
			TenantID: uuid.Nil, // NULL tenant_id in DB = system-level key
			Scopes:   []string{string(permissions.ScopeAdmin)},
		},
	}}

	tenants := newMockTenantStore()
	tenants.addTenant(tenantA, "tenant-a")
	tenants.addTenant(tenantB, "tenant-b")

	// Wire the package globals exactly as server startup does. The gateway
	// token is deliberately unrelated to every API key, so a passing case
	// proves the key path was reached and not the gateway path.
	oldCache, oldTenants, oldGateway := pkgAPIKeyCache, pkgTenantCache, pkgGatewayToken
	InitGatewayToken("fake-gateway-token-never-an-api-key")
	InitAPIKeyCache(keys, nil)
	InitTenantStore(tenants, nil)
	t.Cleanup(func() {
		pkgAPIKeyCache, pkgTenantCache, pkgGatewayToken = oldCache, oldTenants, oldGateway
	})

	tests := []struct {
		name        string
		bearer      string
		tenantHdr   string
		wantAuth    bool
		wantTenant  uuid.UUID
		wantRole    permissions.Role
		wantKeyData bool
	}{
		{
			name:        "tenant-bound key alone resolves to its own tenant",
			bearer:      boundToken,
			wantAuth:    true,
			wantTenant:  tenantA,
			wantRole:    permissions.RoleOperator,
			wantKeyData: true,
		},
		{
			name:        "tenant-bound key ignores header naming another tenant",
			bearer:      boundToken,
			tenantHdr:   "tenant-b",
			wantAuth:    true,
			wantTenant:  tenantA,
			wantRole:    permissions.RoleOperator,
			wantKeyData: true,
		},
		{
			name:        "tenant-bound key with header naming its own tenant stays itself",
			bearer:      boundToken,
			tenantHdr:   "tenant-a",
			wantAuth:    true,
			wantTenant:  tenantA,
			wantRole:    permissions.RoleOperator,
			wantKeyData: true,
		},
		{
			name:        "system-level key is re-scoped by the tenant header",
			bearer:      systemToken,
			tenantHdr:   "tenant-b",
			wantAuth:    true,
			wantTenant:  tenantB,
			wantRole:    permissions.RoleAdmin,
			wantKeyData: true,
		},
		{
			name:     "unresolvable key yields no identity",
			bearer:   unknownKey,
			wantAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/v1/agents", nil)
			r.Header.Set("Authorization", "Bearer "+tt.bearer)
			if tt.tenantHdr != "" {
				r.Header.Set("X-ZalosClaw-Tenant-Id", tt.tenantHdr)
			}

			got := resolveAuthWithBearer(r, tt.bearer)

			if got.Authenticated != tt.wantAuth {
				t.Fatalf("Authenticated = %v, want %v (tenant=%v role=%v)",
					got.Authenticated, tt.wantAuth, got.TenantID, got.Role)
			}
			if !tt.wantAuth {
				if got.TenantID != uuid.Nil || got.KeyData != nil {
					t.Fatalf("unauthenticated result must carry no identity, got tenant=%v key=%v",
						got.TenantID, got.KeyData)
				}
				return
			}
			if (got.KeyData != nil) != tt.wantKeyData {
				t.Fatalf("KeyData non-nil = %v, want %v — the gateway path must not have matched",
					got.KeyData != nil, tt.wantKeyData)
			}
			if got.TenantID != tt.wantTenant {
				t.Fatalf("TenantID = %v, want %v (header %q must not re-scope a tenant-bound key)",
					got.TenantID, tt.wantTenant, tt.tenantHdr)
			}
			if got.Role != tt.wantRole {
				t.Fatalf("Role = %v, want %v", got.Role, tt.wantRole)
			}
		})
	}
}

// fakeAPIKeyStore is the smallest APIKeyStore that can resolve an active key
// by hash. It keeps the test database-free; the package's own
// mockAPIKeyStore (api_key_cache_test.go) is left untouched so this file's
// fixture is explicit.
type fakeAPIKeyStore struct {
	byHash map[string]*store.APIKeyData
}

func (f *fakeAPIKeyStore) GetByHash(_ context.Context, hash string) (*store.APIKeyData, error) {
	return f.byHash[hash], nil
}

func (f *fakeAPIKeyStore) Get(context.Context, uuid.UUID) (*store.APIKeyData, error) {
	return nil, nil
}

func (f *fakeAPIKeyStore) Create(context.Context, *store.APIKeyData) error { return nil }

func (f *fakeAPIKeyStore) List(context.Context, string) ([]store.APIKeyData, error) {
	return nil, nil
}

func (f *fakeAPIKeyStore) Revoke(context.Context, uuid.UUID, string) error { return nil }

func (f *fakeAPIKeyStore) TouchLastUsed(context.Context, uuid.UUID) error { return nil }
