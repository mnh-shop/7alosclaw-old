package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	mcpbridge "github.com/mnh-shop/7alosclaw/internal/mcp"
	"github.com/mnh-shop/7alosclaw/internal/permissions"
	"github.com/mnh-shop/7alosclaw/internal/store"
)

// Sentinel values that must never appear in an MCP server API response.
const (
	secretAPIKey = "SENTINEL-APIKEY-9f3a"
	secretHeader = "SENTINEL-HEADER-b71c"
	secretEnv    = "SENTINEL-ENV-c02d"
)

// scopeTestMCPStore is the in-memory MCPServerStore used by the scope/redaction
// tests. It embeds mockMCPServerForOAuth for the unused interface methods.
type scopeTestMCPStore struct {
	*mockMCPServerForOAuth
	created []*store.MCPServerData
	list    []store.MCPServerData
}

func newScopeTestMCPStore() *scopeTestMCPStore {
	return &scopeTestMCPStore{mockMCPServerForOAuth: newMockMCPServerForOAuth()}
}

func (s *scopeTestMCPStore) CreateServer(_ context.Context, srv *store.MCPServerData) error {
	s.created = append(s.created, srv)
	return nil
}

func (s *scopeTestMCPStore) ListServers(_ context.Context) ([]store.MCPServerData, error) {
	return s.list, nil
}

// tenantAdminRequest returns r with a non-master tenant + admin role in ctx —
// the scope a tenant-level admin actually has after HTTP auth.
func tenantAdminRequest(r *http.Request, tid uuid.UUID) *http.Request {
	ctx := store.WithTenantID(r.Context(), tid)
	ctx = store.WithRole(ctx, string(permissions.RoleAdmin))
	return r.WithContext(ctx)
}

// --- (a) stdio is master/owner-only ---

func TestMCPCreateStdioRejectedForTenantScope(t *testing.T) {
	st := newScopeTestMCPStore()
	h := NewMCPHandler(st, nil, nil)

	body := `{"name":"evil","transport":"stdio","command":"node","args":["server.js"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp/servers", strings.NewReader(body))
	req = tenantAdminRequest(req, uuid.New()) // non-master tenant
	rec := httptest.NewRecorder()
	h.handleCreateServer(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
	}
	if len(st.created) != 0 {
		t.Fatalf("tenant registered %d stdio server(s), want 0", len(st.created))
	}
}

func TestMCPCreateStdioAllowedForMasterScope(t *testing.T) {
	st := newScopeTestMCPStore()
	h := NewMCPHandler(st, nil, nil)

	body := `{"name":"internal","transport":"stdio","command":"node","args":["server.js"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp/servers", strings.NewReader(body))
	// No tenant in ctx → master scope (the pre-existing internal-owner path).
	rec := httptest.NewRecorder()
	h.handleCreateServer(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	if len(st.created) != 1 {
		t.Fatalf("master registered %d stdio server(s), want 1", len(st.created))
	}
}

func TestMCPTestConnectionStdioRejectedForTenantScope(t *testing.T) {
	st := newScopeTestMCPStore()
	h := NewMCPHandler(st, nil, nil)
	discoveryCalls := 0
	h.discoverTools = func(context.Context, string, string, []string, map[string]string, string, map[string]string) ([]mcpbridge.ToolInfo, error) {
		discoveryCalls++
		return nil, nil
	}

	body := `{"transport":"stdio","command":"node","args":["server.js"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp/servers/test", strings.NewReader(body))
	req = tenantAdminRequest(req, uuid.New())
	rec := httptest.NewRecorder()
	h.handleTestConnection(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
	}
	if discoveryCalls != 0 {
		t.Fatalf("tenant stdio config reached discovery %d time(s), want 0", discoveryCalls)
	}
}

// --- (c) secrets are write-only ---

func TestMCPResponsesRedactSecrets(t *testing.T) {
	st := newScopeTestMCPStore()
	server := store.MCPServerData{
		BaseModel: store.BaseModel{ID: uuid.New()},
		Name:      "secrets-srv",
		Transport: "streamable-http",
		URL:       "https://8.8.8.8/mcp",
		APIKey:    secretAPIKey,
		Headers:   json.RawMessage(`{"X-Auth":"` + secretHeader + `"}`),
		Env:       json.RawMessage(`{"TOKEN":"` + secretEnv + `"}`),
	}
	st.list = []store.MCPServerData{server}
	st.servers[server.ID] = &server
	h := NewMCPHandler(st, nil, nil)

	t.Run("list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.handleListServers(rec, httptest.NewRequest(http.MethodGet, "/v1/mcp/servers", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		assertNoMCPSecret(t, rec.Body.String())
		if !strings.Contains(rec.Body.String(), "secrets-srv") {
			t.Fatalf("server name missing from list response: %s", rec.Body.String())
		}
	})

	t.Run("get", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/mcp/servers/"+server.ID.String(), nil)
		req.SetPathValue("id", server.ID.String())
		rec := httptest.NewRecorder()
		h.handleGetServer(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		assertNoMCPSecret(t, rec.Body.String())
	})

	t.Run("create", func(t *testing.T) {
		body := `{"name":"secrets-echo","transport":"streamable-http","url":"https://8.8.8.8/mcp",` +
			`"api_key":"` + secretAPIKey + `","headers":{"X-Auth":"` + secretHeader + `"},"env":{"TOKEN":"` + secretEnv + `"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/mcp/servers", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.handleCreateServer(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
		}
		assertNoMCPSecret(t, rec.Body.String())
	})
}

func assertNoMCPSecret(t *testing.T, body string) {
	t.Helper()
	for _, secret := range []string{secretAPIKey, secretHeader, secretEnv} {
		if strings.Contains(body, secret) {
			t.Fatalf("MCP response leaked %q: %s", secret, body)
		}
	}
}
