package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mnh-shop/7alosclaw/internal/store"
)

// fakeCreateTenantStore embeds the interface and overrides only CreateTenant.
// A slug already taken by an archived tenant must surface as a 409 conflict,
// not a 500 server fault.
type fakeCreateTenantStore struct {
	store.TenantStore
	createErr error
}

func (f *fakeCreateTenantStore) CreateTenant(context.Context, *store.TenantData) error {
	return f.createErr
}

func TestTenantsCreate_DuplicateSlugIsConflict(t *testing.T) {
	const slug = "acme"
	body := `{"name":"Acme","slug":"` + slug + `"}`

	tests := []struct {
		name             string
		createErr        error
		wantStatus       int
		wantBodyContains string
		wantBodyAbsent   string
	}{
		{
			name:             "duplicate key error is a conflict naming the slug",
			createErr:        errors.New(`pq: duplicate key value violates unique constraint "tenants_slug_key"`),
			wantStatus:       http.StatusConflict,
			wantBodyContains: slug,
		},
		{
			name:             "SQLSTATE 23505 alone is a conflict",
			createErr:        errors.New("store: insert failed with SQLSTATE 23505"),
			wantStatus:       http.StatusConflict,
			wantBodyContains: slug,
		},
		{
			name:           "generic store error stays a 500 without the raw text",
			createErr:      errors.New("connection reset"),
			wantStatus:     http.StatusInternalServerError,
			wantBodyAbsent: "connection reset",
		},
		{
			name:       "successful create is 201",
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewTenantsHandler(&fakeCreateTenantStore{createErr: tt.createErr}, nil, "")

			req := httptest.NewRequest(http.MethodPost, "/v1/tenants", strings.NewReader(body))
			req = req.WithContext(store.WithRole(req.Context(), store.RoleOwner))
			rec := httptest.NewRecorder()

			h.handleCreate(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			got := rec.Body.String()
			if tt.wantBodyContains != "" && !strings.Contains(got, tt.wantBodyContains) {
				t.Fatalf("body %q does not contain %q", got, tt.wantBodyContains)
			}
			if tt.wantBodyAbsent != "" && strings.Contains(got, tt.wantBodyAbsent) {
				t.Fatalf("body %q leaked raw store error text %q", got, tt.wantBodyAbsent)
			}
		})
	}
}
