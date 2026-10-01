package http

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/eventbus"
	"github.com/mnh-shop/7alosclaw/internal/store"
)

// scopeFakeWorkstationStore returns a configurable mutation error so the
// handler's RowsAffected -> 404 mapping can be exercised without a database.
type scopeFakeWorkstationStore struct {
	updateErr error
	deleteErr error
}

func (f *scopeFakeWorkstationStore) Create(context.Context, *store.Workstation) error { return nil }
func (f *scopeFakeWorkstationStore) GetByID(context.Context, uuid.UUID) (*store.Workstation, error) {
	return &store.Workstation{BackendType: store.BackendSSH}, nil
}
func (f *scopeFakeWorkstationStore) GetByKey(context.Context, string) (*store.Workstation, error) {
	return nil, sql.ErrNoRows
}
func (f *scopeFakeWorkstationStore) List(context.Context) ([]store.Workstation, error) {
	return nil, nil
}
func (f *scopeFakeWorkstationStore) Update(context.Context, uuid.UUID, map[string]any) error {
	return f.updateErr
}
func (f *scopeFakeWorkstationStore) SetActive(context.Context, uuid.UUID, bool) error { return nil }
func (f *scopeFakeWorkstationStore) Delete(context.Context, uuid.UUID) error          { return f.deleteErr }

// TestWorkstationUpdateDeleteNotFoundStatus is the FIX C transport half: a
// store-level sql.ErrNoRows must become a 404, and a successful mutation a 200.
func TestWorkstationUpdateDeleteNotFoundStatus(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		updateErr  error
		deleteErr  error
		wantStatus int
	}{
		{
			name:       "update nonexistent",
			method:     http.MethodPut,
			body:       `{"name":"x"}`,
			updateErr:  sql.ErrNoRows,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "update cross-tenant",
			method:     http.MethodPut,
			body:       `{"name":"x"}`,
			updateErr:  sql.ErrNoRows,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "update legitimate",
			method:     http.MethodPut,
			body:       `{"name":"x"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete nonexistent",
			method:     http.MethodDelete,
			deleteErr:  sql.ErrNoRows,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete cross-tenant",
			method:     http.MethodDelete,
			deleteErr:  sql.ErrNoRows,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete legitimate",
			method:     http.MethodDelete,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := &scopeFakeWorkstationStore{updateErr: tc.updateErr, deleteErr: tc.deleteErr}
			h := NewWorkstationsHandler(fs, nil, nil)

			id := uuid.New()
			var req *http.Request
			if tc.body == "" {
				req = httptest.NewRequest(tc.method, "/v1/workstations/"+id.String(), nil)
			} else {
				req = httptest.NewRequest(tc.method, "/v1/workstations/"+id.String(), strings.NewReader(tc.body))
			}
			// Owner role bypasses the tenant-membership lookup (nil tenant store).
			req = req.WithContext(store.WithRole(req.Context(), store.RoleOwner))
			req.SetPathValue("id", id.String())
			rec := httptest.NewRecorder()

			if tc.method == http.MethodPut {
				h.handleUpdate(rec, req)
			} else {
				h.handleDelete(rec, req)
			}

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestHTTPUpdatePublishesEvent pins the HTTP half of FIX E: the transport
// handlers are the publishers, so an edit must emit workstation.updated.
func TestHTTPUpdatePublishesEvent(t *testing.T) {
	bus := eventbus.NewDomainEventBus(eventbus.DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)
	defer bus.Drain(time.Second) //nolint:errcheck

	got := make(chan eventbus.DomainEvent, 4)
	bus.Subscribe(eventbus.EventWorkstationUpdated, func(_ context.Context, e eventbus.DomainEvent) error {
		got <- e
		return nil
	})

	h := NewWorkstationsHandler(&scopeFakeWorkstationStore{}, nil, nil)
	h.SetDomainBus(bus)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/v1/workstations/"+id.String(), strings.NewReader(`{"name":"x"}`))
	req = req.WithContext(store.WithRole(req.Context(), store.RoleOwner))
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()
	h.handleUpdate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	select {
	case e := <-got:
		p, ok := e.Payload.(eventbus.WorkstationChangedPayload)
		if !ok {
			t.Fatalf("payload type = %T, want WorkstationChangedPayload", e.Payload)
		}
		if p.WorkstationID != id.String() {
			t.Fatalf("payload workstation id = %q, want %q", p.WorkstationID, id)
		}
	case <-time.After(time.Second):
		t.Fatal("no workstation.updated event published by the HTTP update handler")
	}
}
