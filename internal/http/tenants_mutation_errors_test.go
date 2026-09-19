package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/store"
)

// fakeMutationTenantStore embeds the interface and overrides only the three
// mutation methods exercised below, each returning the same driver-level error.
// Those handlers must log the error, not echo it to the caller.
type fakeMutationTenantStore struct {
	store.TenantStore
	err error
}

func (f *fakeMutationTenantStore) UpdateTenant(context.Context, uuid.UUID, map[string]any) error {
	return f.err
}

func (f *fakeMutationTenantStore) AddUser(context.Context, uuid.UUID, string, string) error {
	return f.err
}

func (f *fakeMutationTenantStore) RemoveUser(context.Context, uuid.UUID, string) error {
	return f.err
}

func TestTenantsMutationErrorsDoNotLeakDriverText(t *testing.T) {
	const tenantID = "00000000-0000-0000-0000-0000000000c1"
	driverErr := errors.New(`pq: relation "tenants" does not exist`)

	tests := []struct {
		name       string
		call       func(h *TenantsHandler, w http.ResponseWriter, r *http.Request)
		method     string
		body       string
		setUserID  bool
		wantStatus int
	}{
		{
			name:       "update returns 500 without the raw error",
			call:       (*TenantsHandler).handleUpdate,
			method:     http.MethodPatch,
			body:       `{"name":"Renamed"}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "users add returns 500 without the raw error",
			call:       (*TenantsHandler).handleUsersAdd,
			method:     http.MethodPost,
			body:       `{"user_id":"user-1","role":"member"}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "users remove returns 500 without the raw error",
			call:       (*TenantsHandler).handleUsersRemove,
			method:     http.MethodDelete,
			setUserID:  true,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewTenantsHandler(&fakeMutationTenantStore{err: driverErr}, nil, "")

			req := httptest.NewRequest(tt.method, "/v1/tenants/"+tenantID, strings.NewReader(tt.body))
			// Handlers parse these before touching the store; without them the
			// request 400s at the guard and never exercises the error path.
			req.SetPathValue("id", tenantID)
			if tt.setUserID {
				req.SetPathValue("userId", "user-1")
			}
			req = req.WithContext(store.WithRole(req.Context(), store.RoleOwner))
			rec := httptest.NewRecorder()

			tt.call(h, rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			got := rec.Body.String()
			for _, leak := range []string{"pq:", "relation", "does not exist"} {
				if strings.Contains(got, leak) {
					t.Fatalf("body %q leaked driver text %q", got, leak)
				}
			}
		})
	}
}
