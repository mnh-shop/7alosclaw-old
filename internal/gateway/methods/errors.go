package methods

import (
	"context"
	"log/slog"

	"github.com/mnh-shop/7alosclaw/internal/gateway"
	"github.com/mnh-shop/7alosclaw/internal/i18n"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/pkg/protocol"
)

// respondStoreError classifies a store/backend failure on the WS surface and
// answers the RPC request without echoing the driver's text to the client. An
// absent row (store.IsNotFound: sql.ErrNoRows or a store not-found sentinel)
// answers NOT_FOUND, naming the entity and the client-supplied identifier;
// any other fault answers INTERNAL with a generic message — the raw error
// stays in the server log. This is the WS counterpart of
// internal/mcp/crud_helpers.go's toolStoreError and consumes the
// machine-readable store contract in internal/store/tenant_store.go.
// Client-caused errors (validation, malformed params, bad uuid) keep their
// own actionable NewErrorResponse calls and never route through here.
func respondStoreError(ctx context.Context, client *gateway.Client, req *protocol.RequestFrame, entity, id string, err error) {
	locale := store.LocaleFromContext(ctx)
	slog.Error("ws method store call failed", "method", req.Method, "entity", entity, "error", err)
	if store.IsNotFound(err) {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrNotFound,
			i18n.T(locale, i18n.MsgNotFound, entity, id)))
		return
	}
	client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInternal,
		i18n.T(locale, i18n.MsgInternalError, entity)))
}
