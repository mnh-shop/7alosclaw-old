package gateway

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/config"
	"github.com/mnh-shop/7alosclaw/internal/crypto"
	httpapi "github.com/mnh-shop/7alosclaw/internal/http"
	"github.com/mnh-shop/7alosclaw/internal/permissions"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/pkg/protocol"
)

// TestWSConnectTenantBoundKeyCannotBeRescoped pins the WS connect-path
// tenant-binding invariant, the sibling of
// internal/http/auth_tenant_binding_test.go (TASKS.md H47).
//
// Source of truth: router.go:195-244 handleConnect. The system-key branch
// (217-224) applies params.TenantID via applyTenantScope; the tenant-bound
// branch (233-240) sets client.tenantID = keyData.TenantID and never reads
// params.TenantID. The table includes the one path that MAY re-scope (a
// system-level key, tenant_id NULL) so the asymmetry is stated, not hidden.
//
// The connect path is reachable directly through router.Handle with a nil
// websocket conn (NewClient tolerates nil; SendResponse is non-blocking), so
// this is a unit test, not a websocket harness.
func TestWSConnectTenantBoundKeyCannotBeRescoped(t *testing.T) {
	tenantA, tenantB := uuid.MustParse("00000000-0000-4000-8000-0000000000a1"), uuid.MustParse("00000000-0000-4000-8000-0000000000b2")

	const (
		boundToken  = "ws-api-key-bound-to-tenant-a"
		systemToken = "ws-api-key-system-level"
		unknownKey  = "ws-api-key-that-was-revoked"
	)

	keys := &wsAPIKeyStore{byHash: map[string]*store.APIKeyData{
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

	tenants := newWSTenantStore()
	tenants.addTenant(tenantA, "tenant-a")
	tenants.addTenant(tenantB, "tenant-b")

	// The gateway token is deliberately unrelated to every API key, so a
	// passing case proves the API-key path authenticated, not Path 1.
	cfg := config.Default()
	cfg.Gateway.Token = "fake-gateway-token-never-an-api-key"

	// ResolveAPIKey reads the http package's package-level cache; there is no
	// exported reset, so cleanup swaps in an empty store (behaviourally the
	// same as an uninitialised cache for any later unknown token).
	httpapi.InitAPIKeyCache(keys, nil)
	t.Cleanup(func() {
		httpapi.InitAPIKeyCache(&wsAPIKeyStore{byHash: map[string]*store.APIKeyData{}}, nil)
	})

	server := NewServer(cfg, nil, nil, nil)
	server.router.SetTenantStore(tenants)

	tests := []struct {
		name        string
		token       string
		tenantParam string
		wantAuth    bool
		wantTenant  uuid.UUID
		wantRole    permissions.Role
	}{
		{
			name:       "tenant-bound key alone binds to its own tenant",
			token:      boundToken,
			wantAuth:   true,
			wantTenant: tenantA,
			wantRole:   permissions.RoleOperator,
		},
		{
			name:        "tenant-bound key ignores tenant_id naming another tenant",
			token:       boundToken,
			tenantParam: "tenant-b",
			wantAuth:    true,
			wantTenant:  tenantA,
			wantRole:    permissions.RoleOperator,
		},
		{
			name:        "tenant-bound key with tenant_id naming its own tenant stays itself",
			token:       boundToken,
			tenantParam: "tenant-a",
			wantAuth:    true,
			wantTenant:  tenantA,
			wantRole:    permissions.RoleOperator,
		},
		{
			name:        "system-level key is scoped by tenant_id",
			token:       systemToken,
			tenantParam: "tenant-b",
			wantAuth:    true,
			wantTenant:  tenantB,
			wantRole:    permissions.RoleAdmin,
		},
		{
			name:     "unresolvable key yields no authenticated client",
			token:    unknownKey,
			wantAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{
				"token":     tt.token,
				"user_id":   "caller-1",
				"tenant_id": tt.tenantParam,
			})
			if err != nil {
				t.Fatalf("marshal connect params: %v", err)
			}
			req := &protocol.RequestFrame{
				ID:     "req-1",
				Method: protocol.MethodConnect,
				Params: raw,
			}
			client := NewClient(nil, server, "203.0.113.10")

			server.router.Handle(context.Background(), client, req)

			if client.authenticated != tt.wantAuth {
				t.Fatalf("authenticated = %v, want %v (tenant=%v role=%v)",
					client.authenticated, tt.wantAuth, client.TenantID(), client.Role())
			}
			if !tt.wantAuth {
				if client.TenantID() != uuid.Nil || client.Role() != "" {
					t.Fatalf("unauthenticated client must carry no identity, got tenant=%v role=%v",
						client.TenantID(), client.Role())
				}
				return
			}
			if client.TenantID() != tt.wantTenant {
				t.Fatalf("tenantID = %v, want %v (tenant_id param %q must not re-scope a tenant-bound key)",
					client.TenantID(), tt.wantTenant, tt.tenantParam)
			}
			if client.Role() != tt.wantRole {
				t.Fatalf("role = %v, want %v", client.Role(), tt.wantRole)
			}
		})
	}
}

// wsAPIKeyStore embeds the interface and implements only what ResolveAPIKey
// reaches: GetByHash for the lookup and TouchLastUsed for the async touch.
// Same shape as wsTenantStore in router_tenant_hint_test.go.
type wsAPIKeyStore struct {
	store.APIKeyStore
	byHash map[string]*store.APIKeyData
}

func (m *wsAPIKeyStore) GetByHash(_ context.Context, hash string) (*store.APIKeyData, error) {
	return m.byHash[hash], nil
}

func (m *wsAPIKeyStore) TouchLastUsed(context.Context, uuid.UUID) error { return nil }

type wsTenantStore struct {
	store.TenantStore
	bySlug map[string]*store.TenantData
}

func newWSTenantStore() *wsTenantStore { return &wsTenantStore{bySlug: make(map[string]*store.TenantData)} }

func (m *wsTenantStore) addTenant(id uuid.UUID, slug string) {
	m.bySlug[slug] = &store.TenantData{ID: id, Slug: slug}
}

func (m *wsTenantStore) GetTenantBySlug(_ context.Context, slug string) (*store.TenantData, error) {
	return m.bySlug[slug], nil
}

func (m *wsTenantStore) GetTenant(_ context.Context, id uuid.UUID) (*store.TenantData, error) {
	for _, tenant := range m.bySlug {
		if tenant.ID == id {
			return tenant, nil
		}
	}
	return nil, nil
}

func (m *wsTenantStore) GetUserRole(context.Context, uuid.UUID, string) (string, error) {
	return store.TenantRoleAdmin, nil
}
