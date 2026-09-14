package http

import (
	"context"
	"encoding/json"
	"errors"
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

// Regression for the fail-open guard: an admin with an absent tenant context is
// treated as master by store.IsMasterScope (uuid.Nil fallback) but must fail
// closed under store.IsExplicitMasterScope. Fails if reverted to IsMasterScope.
func TestAgentUpdateSandboxConfigMissingTenantFailsClosed(t *testing.T) {
	status, stub := runAgentSandboxUpdate(t, uuid.Nil, permissions.RoleAdmin)

	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
	if stub.updated != nil {
		t.Fatalf("sandbox_config update must not reach the store, got %v", stub.updated)
	}
}

// ---- POST /v1/agents (create) sandbox_config scope ----

// runAgentSandboxCreate exercises handleCreate with the full AgentData body.
// gatewayOperatorAgentStore is reused as a capturing AgentStore.
func runAgentSandboxCreate(t *testing.T, tenantID uuid.UUID, role permissions.Role, withSandbox bool) (int, *gatewayOperatorAgentStore) {
	t.Helper()
	stub := &gatewayOperatorAgentStore{}
	h := &AgentsHandler{agents: stub, defaultWorkspace: t.TempDir()}

	body := `{"agent_key":"sandbox-create","display_name":"Sandbox Create","provider":"anthropic","model":"claude"}`
	if withSandbox {
		body = `{"agent_key":"sandbox-create","display_name":"Sandbox Create","provider":"anthropic","model":"claude","sandbox_config":{"network":true}}`
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/agents", strings.NewReader(body))
	ctx := store.WithUserID(context.Background(), "creator")
	if tenantID != uuid.Nil {
		ctx = store.WithTenantID(ctx, tenantID)
	}
	ctx = store.WithRole(ctx, string(role))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.handleCreate(rec, req)
	return rec.Code, stub
}

func TestAgentCreateSandboxConfigTenantRejected(t *testing.T) {
	status, stub := runAgentSandboxCreate(t, uuid.New(), permissions.RoleAdmin, true)

	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
	if len(stub.agents) != 0 {
		t.Fatalf("agent must not be created for tenant sandbox_config, got %+v", stub.agents)
	}
}

func TestAgentCreateSandboxConfigMasterAllowed(t *testing.T) {
	status, stub := runAgentSandboxCreate(t, store.MasterTenantID, permissions.RoleOwner, true)

	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", status)
	}
	if len(stub.agents) != 1 {
		t.Fatalf("expected one created agent, got %+v", stub.agents)
	}
	if len(stub.agents[0].SandboxConfig) == 0 {
		t.Fatalf("expected sandbox_config to be persisted, got %+v", stub.agents[0])
	}
}

func TestAgentCreateSandboxConfigMissingTenantFailsClosed(t *testing.T) {
	status, stub := runAgentSandboxCreate(t, uuid.Nil, permissions.RoleAdmin, true)

	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
	if len(stub.agents) != 0 {
		t.Fatalf("agent must not be created for missing-tenant sandbox_config, got %+v", stub.agents)
	}
}

// ---- import sandbox_config scope ----

// importSandboxAgentStore captures the AgentData handed to Create and aborts so
// doImportNewAgent returns before doMergeImport needs the rest of the store.
type importSandboxAgentStore struct {
	store.AgentStore
	created *store.AgentData
}

func (s *importSandboxAgentStore) GetByKey(_ context.Context, _ string) (*store.AgentData, error) {
	return nil, nil
}

func (s *importSandboxAgentStore) Create(_ context.Context, d *store.AgentData) error {
	s.created = d
	return errors.New("stop after capture")
}

func runImportSandboxConfig(t *testing.T, role permissions.Role) *store.AgentData {
	t.Helper()
	stub := &importSandboxAgentStore{}
	h := &AgentsHandler{agents: stub, defaultWorkspace: t.TempDir()}

	arc := &importArchive{
		agentConfig: map[string]json.RawMessage{
			"agent_key":      json.RawMessage(`"imported-agent"`),
			"sandbox_config": json.RawMessage(`{"network":true}`),
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/agents/import", nil)
	// No tenant in ctx (uuid.Nil) — IsMasterScope would pass, explicit must strip.
	ctx := store.WithRole(context.Background(), string(role))
	if _, err := h.doImportNewAgent(ctx, req, arc, nil); err == nil {
		t.Fatal("expected Create error to abort import after capture")
	}
	if stub.created == nil {
		t.Fatal("agentStore.Create was not called")
	}
	return stub.created
}

func TestImportSandboxConfigMissingTenantStripped(t *testing.T) {
	created := runImportSandboxConfig(t, permissions.RoleAdmin)

	if len(created.SandboxConfig) != 0 {
		t.Fatalf("sandbox_config must be stripped for missing tenant, got %s", created.SandboxConfig)
	}
}

func TestImportSandboxConfigMasterAllowed(t *testing.T) {
	created := runImportSandboxConfig(t, permissions.RoleOwner)

	if len(created.SandboxConfig) == 0 {
		t.Fatalf("sandbox_config must be preserved for owner, got %+v", created)
	}
}
