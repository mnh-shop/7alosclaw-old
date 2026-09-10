package http

import (
	"net/http"

	"github.com/nextlevelbuilder/zalosclaw/internal/i18n"
	"github.com/nextlevelbuilder/zalosclaw/pkg/protocol"
)

// RegisterAPINotFoundRoute keeps unknown /v1/* paths in the structured API envelope.
func RegisterAPINotFoundRoute(mux *http.ServeMux) {
	mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, protocol.ErrNotFound, i18n.T(extractLocale(r), i18n.MsgNotFound, "API route", r.URL.Path))
	})
}
