package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/permissions"
	"github.com/mnh-shop/7alosclaw/internal/store"
)

// sandboxAgentStore is a minimal AgentStore for the PUT /v1/agents/{id}
// sandbox_config scope tests. Embedding store.AgentStore satisfies the interface;
// only GetByID and Update are exercised.
type sandboxAgentStore struct {
	store.AgentStore
	tenantID uuid.UUID
	updated  map[string]any
}

func (s *sandboxAgentStore) GetByID(_ context.Context, id uuid.UUID) (*store.AgentData, error) {
	return &store.AgentData{
		BaseModel: store.BaseModel{ID: id},
		AgentKey:  "sandbox-agent",
		TenantID:  s.tenantID,
	}, nil
}

func (s *sandboxAgentStore) Update(_ context.Context, _ uuid.UUID, updates map[string]any) error {
	s.updated = updates
	return nil
}

func runAgentSandboxUpdate(t *testing.T, tenantID uuid.UUID, role permissions.Role) (int, *sandboxAgentStore) {
	t.Helper()
	stub := &sandboxAgentStore{tenantID: tenantID}
	h := &AgentsHandler{agents: stub}

	agentID := uuid.New()
	body := `{"sandbox_config":{"network":true,"read_only_root":false}}`
	req := httptest.NewRequest(http.MethodPut, "/v1/agents/"+agentID.String(), strings.NewReader(body))
	req.SetPathValue("id", agentID.String())
	ctx := context.Background()
	if tenantID != uuid.Nil {
		ctx = store.WithTenantID(ctx, tenantID)
	}
	ctx = store.WithRole(ctx, string(role))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.handleUpdate(rec, req)
	return rec.Code, stub
}

func TestAgentUpdateSandboxConfigTenantRejected(t *testing.T) {
	tenantID := uuid.New() // non-master
	status, stub := runAgentSandboxUpdate(t, tenantID, permissions.RoleAdmin)

	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
	if stub.updated != nil {
		t.Fatalf("sandbox_config update must not reach the store, got %v", stub.updated)
	}
}

func TestAgentUpdateSandboxConfigMasterAllowed(t *testing.T) {
	status, stub := runAgentSandboxUpdate(t, uuid.Nil, permissions.RoleOwner)

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if stub.updated["sandbox_config"] == nil {
		t.Fatalf("expected sandbox_config to be persisted, got %v", stub.updated)
	}
}
