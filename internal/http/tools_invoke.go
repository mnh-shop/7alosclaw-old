package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mnh-shop/7alosclaw/internal/config"
	"github.com/mnh-shop/7alosclaw/internal/i18n"
	"github.com/mnh-shop/7alosclaw/internal/permissions"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/internal/tools"
)

// ToolsInvokeHandler handles POST /v1/tools/invoke (direct tool invocation).
type ToolsInvokeHandler struct {
	registry      *tools.Registry
	agentStore    store.AgentStore // nil if not configured
	workspaceBase string           // global workspace root, for tenant-scoped sandbox mounts
}

// NewToolsInvokeHandler creates a handler for the tools invoke endpoint.
func NewToolsInvokeHandler(registry *tools.Registry, agentStore store.AgentStore, workspaceBase string) *ToolsInvokeHandler {
	return &ToolsInvokeHandler{
		registry:      registry,
		agentStore:    agentStore,
		workspaceBase: workspaceBase,
	}
}

type toolsInvokeRequest struct {
	Tool       string         `json:"tool"`
	Action     string         `json:"action,omitempty"`
	Args       map[string]any `json:"args"`
	SessionKey string         `json:"sessionKey,omitempty"`
	AgentID    string         `json:"agentId,omitempty"`
	DryRun     bool           `json:"dryRun,omitempty"`
	Channel    string         `json:"channel,omitempty"`  // tool context: channel name
	ChatID     string         `json:"chatId,omitempty"`   // tool context: chat ID
	PeerKind   string         `json:"peerKind,omitempty"` // tool context: "direct" or "group"
}

func (h *ToolsInvokeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	locale := extractLocale(r)

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": i18n.T(locale, i18n.MsgMethodNotAllowed)})
		return
	}

	auth := resolveAuth(r)
	if !auth.Authenticated {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": i18n.T(locale, i18n.MsgUnauthorized)})
		return
	}
	if !permissions.HasMinRole(auth.Role, permissions.RoleOperator) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": i18n.T(locale, i18n.MsgPermissionDenied, r.URL.Path)})
		return
	}

	// Inject tenant, role, user, and locale into context for downstream stores/tools.
	r = r.WithContext(enrichContext(r.Context(), r, auth))

	var req toolsInvokeRequest
	if !bindJSON(w, r, locale, &req) {
		return
	}

	if req.Tool == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": i18n.T(locale, i18n.MsgRequired, "tool")})
		return
	}

	slog.Info("tools invoke request", "tool", req.Tool, "dry_run", req.DryRun)

	if req.DryRun {
		// Just check if tool exists and return its schema
		tool, ok := h.registry.Get(req.Tool)
		if !ok {
			writeToolError(w, http.StatusNotFound, "NOT_FOUND", i18n.T(locale, i18n.MsgNotFound, "tool", req.Tool))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"tool":        req.Tool,
			"description": tool.Description(),
			"parameters":  tool.Parameters(),
			"dryRun":      true,
		})
		return
	}

	// Inject agentID into context for interceptors (bootstrap, memory).
	// Note: userID, tenantID, role, locale already injected by enrichContext above.
	ctx := r.Context()

	agentIDStr := req.AgentID
	if agentIDStr == "" {
		agentIDStr = extractAgentID(r, "")
	}
	if agentIDStr != "" && h.agentStore != nil {
		ag, err := h.agentStore.GetByKey(ctx, agentIDStr)
		if err == nil {
			ctx = store.WithAgentID(ctx, ag.ID)
		}
	}

	// Inject tool context keys (channel, chatID, peerKind) for message routing.
	if req.Channel != "" {
		ctx = tools.WithToolChannel(ctx, req.Channel)
	}
	if req.ChatID != "" {
		ctx = tools.WithToolChatID(ctx, req.ChatID)
	}
	if req.PeerKind != "" {
		ctx = tools.WithToolPeerKind(ctx, req.PeerKind)
	}

	// Execute the tool
	args := req.Args
	if args == nil {
		args = make(map[string]any)
	}

	// If action is specified, add it to args
	if req.Action != "" {
		args["action"] = req.Action
	}

	// Sandbox routing: derive a tenant-scoped session key and mount the tenant
	// workspace so sandbox-aware tools run in a per-session sandbox instead of
	// the host. With sandboxing enabled, exec fails closed without these.
	sessionKey := req.SessionKey
	if sessionKey == "" {
		sessionKey = "http:" + store.TenantIDFromContext(ctx).String()
	}
	if h.workspaceBase != "" && !store.IsMasterScope(ctx) {
		ctx = tools.WithToolWorkspace(ctx, config.TenantWorkspace(
			h.workspaceBase,
			store.TenantIDFromContext(ctx),
			store.TenantSlugFromContext(ctx),
		))
	}

	result := h.registry.ExecuteWithContext(ctx, req.Tool, args, "http", "api", "direct", sessionKey, nil)

	if result.IsError {
		writeToolError(w, http.StatusBadRequest, "TOOL_ERROR", result.ForLLM)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"result": map[string]any{
			"output":   result.ForLLM,
			"forUser":  result.ForUser,
			"metadata": map[string]any{},
		},
	})
}

func writeToolError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
