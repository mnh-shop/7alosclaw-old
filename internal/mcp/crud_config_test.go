package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mnh-shop/7alosclaw/internal/config"
)

func TestConfigGet_ReturnsMaskedCopy(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.Token = "super-secret-token"
	srv := newTestMCPServer()
	registerConfigCRUDTools(srv, cfg)

	result := callTool(t, srv, "zalosclaw_config_get", map[string]any{})
	require.False(t, toolIsError(result))
	// MaskedCopy must never leak the raw token verbatim.
	assert.NotContains(t, toolResultText(result), "super-secret-token")
}
