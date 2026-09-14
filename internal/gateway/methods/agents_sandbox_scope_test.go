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
