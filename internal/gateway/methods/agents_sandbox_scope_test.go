package methods

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/gateway"
	"github.com/mnh-shop/7alosclaw/internal/permissions"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/pkg/protocol"
)

// sandboxScopeStore is a minimal AgentStore for agents.update scope tests. It
// embeds createCaptureStore to satisfy the interface and overrides only the
// two methods agents.update touches before/at the guard.
type sandboxScopeStore struct {
	*createCaptureStore
	agent   *store.AgentData
	updates map[string]any
}

func (s *sandboxScopeStore) GetByKey(_ context.Context, _ string) (*store.AgentData, error) {
	return s.agent, nil
}

func (s *sandboxScopeStore) Update(_ context.Context, _ uuid.UUID, updates map[string]any) error {
	s.updates = updates
	return nil
}

func newSandboxScopeStore() *sandboxScopeStore {
	return &sandboxScopeStore{
		createCaptureStore: &createCaptureStore{},
		agent: &store.AgentData{
			BaseModel: store.BaseModel{ID: uuid.New()},
			AgentKey:  "sandbox-agent",
			AgentType: store.AgentTypePredefined,
		},
	}
}

func sandboxUpdateRequest(t *testing.T, sandbox map[string]any) *protocol.RequestFrame {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"agentId":        "sandbox-agent",
		"sandbox_config": sandbox,
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	return &protocol.RequestFrame{
		Type:   protocol.FrameTypeRequest,
		ID:     "sandbox-update-1",
		Method: protocol.MethodAgentsUpdate,
		Params: raw,
	}
}

func readSandboxResponse(t *testing.T, responses <-chan []byte) protocol.ResponseFrame {
	t.Helper()
	select {
	case raw := <-responses:
		var resp protocol.ResponseFrame
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		return resp
	default:
		t.Fatal("expected a response frame, got none")
		return protocol.ResponseFrame{}
	}
}

// A tenant-scope (non-master) caller must not be able to rewrite sandbox
// containment via agents.update.
func TestWSUpdateSandboxConfigTenantRejected(t *testing.T) {
	stub := newSandboxScopeStore()
	m := newManagedMethods(t, stub)
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	client, responses := gateway.NewCapturingTestClient(permissions.RoleAdmin, tenantID, "tenant-admin", 4)

	m.handleUpdate(wsCallCtx(client), client, sandboxUpdateRequest(t, map[string]any{"network": true}))

	resp := readSandboxResponse(t, responses)
	if resp.Error == nil {
		t.Fatalf("expected authorization error, got ok=%v payload=%v", resp.OK, resp.Payload)
	}
	if stub.updates != nil {
		t.Fatalf("sandbox_config update must not reach the store, got %v", stub.updates)
	}
}

// Master/owner scope may still set sandbox_config.
func TestWSUpdateSandboxConfigMasterAllowed(t *testing.T) {
	stub := newSandboxScopeStore()
	m := newManagedMethods(t, stub)
	client, responses := gateway.NewCapturingTestClient(permissions.RoleOwner, store.MasterTenantID, "owner", 4)

	m.handleUpdate(wsCallCtx(client), client, sandboxUpdateRequest(t, map[string]any{"network": true}))

	resp := readSandboxResponse(t, responses)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if stub.updates["sandbox_config"] == nil {
		t.Fatalf("expected sandbox_config to be persisted, got %v", stub.updates)
	}
}

// Regression for the fail-open guard: a non-owner caller with an absent tenant
// context is treated as master by store.IsMasterScope (uuid.Nil fallback) but
// must fail closed under store.IsExplicitMasterScope. This test fails if the
// handler is reverted to IsMasterScope.
func TestWSUpdateSandboxConfigMissingTenantFailsClosed(t *testing.T) {
	stub := newSandboxScopeStore()
	m := newManagedMethods(t, stub)
	// uuid.Nil tenant + admin role => no explicit master scope.
	client, responses := gateway.NewCapturingTestClient(permissions.RoleAdmin, uuid.Nil, "no-tenant-admin", 4)

	m.handleUpdate(wsCallCtx(client), client, sandboxUpdateRequest(t, map[string]any{"network": true}))

	resp := readSandboxResponse(t, responses)
	if resp.Error == nil {
		t.Fatalf("missing-tenant ctx must fail closed, got ok payload=%v", resp.Payload)
	}
	if stub.updates != nil {
		t.Fatalf("sandbox_config update must not reach the store, got %v", stub.updates)
	}
}

// A tenant-scope caller must not be able to set sandbox containment at agent
// creation over the WS agents.create method.
func TestWSCreateSandboxConfigTenantRejected(t *testing.T) {
	stub := &createCaptureStore{}
	m := newManagedMethods(t, stub)
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	client, responses := gateway.NewCapturingTestClient(permissions.RoleAdmin, tenantID, "tenant-admin", 4)

	req := buildCreateRequest(t, map[string]any{
		"name":           "Tenant Sandbox Agent",
		"sandbox_config": map[string]any{"network": true},
	})
	m.handleCreate(wsCallCtx(client), client, req)

	resp := readSandboxResponse(t, responses)
	if resp.Error == nil {
		t.Fatalf("expected authorization error, got ok=%v payload=%v", resp.OK, resp.Payload)
	}
	if stub.created != nil {
		t.Fatalf("agent must not be created for tenant sandbox_config, got %+v", stub.created)
	}
}

// Master/owner scope may still set sandbox_config at agent creation.
func TestWSCreateSandboxConfigMasterAllowed(t *testing.T) {
	stub := &createCaptureStore{}
	m := newManagedMethods(t, stub)
	client, responses := gateway.NewCapturingTestClient(permissions.RoleOwner, store.MasterTenantID, "owner", 4)

	req := buildCreateRequest(t, map[string]any{
		"name":           "Master Sandbox Agent",
		"sandbox_config": map[string]any{"network": true},
	})
	m.handleCreate(wsCallCtx(client), client, req)

	resp := readSandboxResponse(t, responses)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if stub.created == nil {
		t.Fatal("agentStore.Create was not called")
	}
	if stub.created.SandboxConfig == nil {
		t.Fatalf("expected sandbox_config to be persisted, got %+v", stub.created)
	}
}
