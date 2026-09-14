package mcp

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/store"
)

// resolveServerCredentials is the single shared connect boundary (both
// LoadForAgent's persistent connect and ListToolsForAgent's on-demand
// discovery pass through it), so these tests exercise the tenant guard there.

func tenantGuardCtx(id uuid.UUID) context.Context {
	return store.WithTenantID(context.Background(), id)
}

func plainTenantServer(tenantID uuid.UUID) store.MCPServerData {
	return store.MCPServerData{
		BaseModel: store.BaseModel{ID: uuid.New()},
		Name:      "plain-srv",
		Transport: "streamable-http",
		URL:       "http://127.0.0.1:1/mcp",
		Enabled:   true,
		TenantID:  tenantID,
	}
}

// A tenant must not be able to connect a server owned by another tenant.
func TestServerTenantAllowed_CrossTenantDenied(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	info := store.MCPAccessInfo{Server: plainTenantServer(tenantB)}

	m := &Manager{}
	if rs := m.resolveServerCredentials(tenantGuardCtx(tenantA), info, "user-1"); rs != nil {
		t.Fatalf("expected nil for cross-tenant server, got %+v", rs)
	}
}

// Same-tenant path is unaffected.
func TestServerTenantAllowed_SameTenantAllowed(t *testing.T) {
	tenant := uuid.New()
	info := store.MCPAccessInfo{Server: plainTenantServer(tenant)}

	m := &Manager{}
	if rs := m.resolveServerCredentials(tenantGuardCtx(tenant), info, "user-1"); rs == nil {
		t.Fatal("expected same-tenant server to resolve")
	}
}

// Master scope still connects master-owned servers.
func TestServerTenantAllowed_MasterScopeConnectsMasterServer(t *testing.T) {
	ctx := store.WithTenantID(context.Background(), store.MasterTenantID)
	info := store.MCPAccessInfo{Server: plainTenantServer(store.MasterTenantID)}

	m := &Manager{}
	if rs := m.resolveServerCredentials(ctx, info, ""); rs == nil {
		t.Fatal("expected master scope to connect master-owned server")
	}
}

// Master-owned servers stay usable by tenants (legitimate shared servers).
func TestServerTenantAllowed_TenantCanUseMasterOwnedServer(t *testing.T) {
	info := store.MCPAccessInfo{Server: plainTenantServer(store.MasterTenantID)}

	m := &Manager{}
	if rs := m.resolveServerCredentials(tenantGuardCtx(uuid.New()), info, "user-1"); rs == nil {
		t.Fatal("expected master-owned server to be usable by a tenant")
	}
}

// A lost/nil tenant context fails closed instead of resolving to global scope.
func TestServerTenantAllowed_MissingTenantFailsClosed(t *testing.T) {
	info := store.MCPAccessInfo{Server: plainTenantServer(uuid.New())}

	m := &Manager{}
	if rs := m.resolveServerCredentials(context.Background(), info, "user-1"); rs != nil {
		t.Fatal("expected nil (fail closed) when tenant context is missing")
	}
}

// A server whose owning tenant is unknown fails closed.
func TestServerTenantAllowed_UnknownServerTenantFailsClosed(t *testing.T) {
	info := store.MCPAccessInfo{Server: plainTenantServer(uuid.Nil)}

	m := &Manager{}
	if rs := m.resolveServerCredentials(tenantGuardCtx(uuid.New()), info, "user-1"); rs != nil {
		t.Fatal("expected nil (fail closed) when server tenant is unknown")
	}
}
