package mcp

import (
	"context"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/mnh-shop/7alosclaw/internal/config"
)

// registerConfigCRUDTools registers the zalosclaw_config_get MCP tool. Config is
// read-only through this surface — mutating live gateway config from an MCP
// tool call is out of scope; use the existing config.patch WS method / HTTP
// admin API for writes, which enforce permission and validation rules this
// server does not duplicate.
func registerConfigCRUDTools(srv *mcpserver.MCPServer, cfg *config.Config) {
	srv.AddTool(mcpgo.NewTool("zalosclaw_config_get",
		mcpgo.WithDescription("Get the current gateway configuration, with all secrets masked."),
		mcpgo.WithReadOnlyHintAnnotation(true),
	), handleConfigGet(cfg))
}

func handleConfigGet(cfg *config.Config) mcpserver.ToolHandlerFunc {
	return func(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		return jsonToolResult(cfg.MaskedCopy())
	}
}
