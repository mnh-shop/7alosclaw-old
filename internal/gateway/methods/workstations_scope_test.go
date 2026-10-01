package methods

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/eventbus"
	"github.com/mnh-shop/7alosclaw/internal/gateway"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/pkg/protocol"
)

// --- fakes ---

type fakeLinkStore struct {
	linkErr error
}

func (f *fakeLinkStore) Link(context.Context, *store.AgentWorkstationLink) error { return f.linkErr }
func (f *fakeLinkStore) Unlink(context.Context, uuid.UUID, uuid.UUID) error      { return nil }
func (f *fakeLinkStore) SetDefault(context.Context, uuid.UUID, uuid.UUID) error  { return nil }
func (f *fakeLinkStore) ListForAgent(context.Context, uuid.UUID) ([]store.AgentWorkstationLink, error) {
	return nil, nil
}
func (f *fakeLinkStore) ListForWorkstation(context.Context, uuid.UUID) ([]store.AgentWorkstationLink, error) {
	return nil, nil
}

type fakeWSStore struct {
	updateErr error
	deleteErr error
	createErr error
}

func (f *fakeWSStore) Create(_ context.Context, ws *store.Workstation) error {
	if ws.ID == uuid.Nil {
		ws.ID = uuid.New()
	}
	return f.createErr
}
func (f *fakeWSStore) GetByID(_ context.Context, id uuid.UUID) (*store.Workstation, error) {
	return &store.Workstation{ID: id, BackendType: store.BackendDocker}, nil
}
func (f *fakeWSStore) GetByKey(context.Context, string) (*store.Workstation, error) {
	return nil, sql.ErrNoRows
}
func (f *fakeWSStore) List(context.Context) ([]store.Workstation, error) { return nil, nil }
func (f *fakeWSStore) Update(context.Context, uuid.UUID, map[string]any) error {
	return f.updateErr
}
func (f *fakeWSStore) SetActive(context.Context, uuid.UUID, bool) error { return nil }
func (f *fakeWSStore) Delete(context.Context, uuid.UUID) error          { return f.deleteErr }

type fakePermStore struct {
	getErr error
	perm   *store.WorkstationPermission
}

func (f *fakePermStore) ListForWorkstation(context.Context, uuid.UUID) ([]store.WorkstationPermission, error) {
	return nil, nil
}
func (f *fakePermStore) GetByID(context.Context, uuid.UUID) (*store.WorkstationPermission, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.perm, nil
}
func (f *fakePermStore) Add(context.Context, *store.WorkstationPermission) error { return nil }
func (f *fakePermStore) Remove(context.Context, uuid.UUID) error                 { return nil }
func (f *fakePermStore) SetEnabled(context.Context, uuid.UUID, bool) error       { return nil }
func (f *fakePermStore) SeedDefaults(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

// callWSHandler runs h with a background context and returns the first
// response frame the handler sends.
func callWSHandler(t *testing.T, h gateway.MethodHandler, params any) *protocol.ResponseFrame {
	t.Helper()
	return callWSHandlerCtx(t, context.Background(), h, params)
}

// callWSHandlerCtx is callWSHandler with an explicit context. A live client is
// required because a zero-value gateway.Client has a nil send channel;
// NewClient allocates it.
func callWSHandlerCtx(t *testing.T, ctx context.Context, h gateway.MethodHandler, params any) *protocol.ResponseFrame {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	req := &protocol.RequestFrame{Type: protocol.FrameTypeRequest, ID: "req-1", Method: "test", Params: raw}
	client := gateway.NewClient(nil, nil, "127.0.0.1")
	h(ctx, client, req)

	select {
	case out := <-client.OutboundForTest():
		var resp protocol.ResponseFrame
		if err := json.Unmarshal(out, &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		return &resp
	case <-time.After(time.Second):
		t.Fatal("handler sent no response")
		return nil
	}
}

func wantErrCode(t *testing.T, resp *protocol.ResponseFrame, code string) {
	t.Helper()
	if resp.OK || resp.Error == nil {
		t.Fatalf("response = %+v, want error %s", resp, code)
	}
	if resp.Error.Code != code {
		t.Fatalf("error code = %s, want %s (message %q)", resp.Error.Code, code, resp.Error.Message)
	}
}

// --- FIX A: WS linkAgent path ---

func TestWSLinkAgentErrorMapping(t *testing.T) {
	tests := []struct {
		name    string
		linkErr error
		wantErr string // empty means success expected
	}{
		{name: "cross-tenant or unknown is not found", linkErr: sql.ErrNoRows, wantErr: protocol.ErrNotFound},
		{name: "default conflict is a clean already-exists", linkErr: store.ErrAgentDefaultConflict, wantErr: protocol.ErrAlreadyExists},
		{name: "legitimate same-tenant link succeeds", linkErr: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewWorkstationsMethods(&fakeWSStore{}, &fakeLinkStore{linkErr: tc.linkErr})
			resp := callWSHandler(t, m.handleLinkAgent, map[string]any{
				"agentId":       uuid.NewString(),
				"workstationId": uuid.NewString(),
				"isDefault":     true,
			})
			if tc.wantErr == "" {
				if !resp.OK {
					t.Fatalf("response = %+v, want success", resp)
				}
				return
			}
			wantErrCode(t, resp, tc.wantErr)
			// A cross-tenant failure must not leak the driver's constraint text.
			if errors.Is(tc.linkErr, store.ErrAgentDefaultConflict) && resp.Error.Message == tc.linkErr.Error() {
				t.Fatalf("conflict message leaks the raw error: %q", resp.Error.Message)
			}
		})
	}
}

// --- FIX C: WS update/delete mapping ---

func TestWSWorkstationUpdateDeleteErrorMapping(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name    string
		handler func(*WorkstationsMethods) gateway.MethodHandler
		params  any
		err     error
		setErr  func(*WorkstationsMethods, error)
		wantErr string
	}{
		{
			name:    "update nonexistent",
			handler: func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleUpdate },
			params:  map[string]any{"id": id, "updates": map[string]any{"name": "x"}},
			err:     sql.ErrNoRows,
			setErr:  func(m *WorkstationsMethods, e error) { m.wsStore = &fakeWSStore{updateErr: e} },
			wantErr: protocol.ErrNotFound,
		},
		{
			name:    "update legitimate",
			handler: func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleUpdate },
			params:  map[string]any{"id": id, "updates": map[string]any{"name": "x"}},
			setErr:  func(m *WorkstationsMethods, e error) { m.wsStore = &fakeWSStore{updateErr: e} },
		},
		{
			name:    "delete nonexistent",
			handler: func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleDelete },
			params:  map[string]any{"id": id},
			err:     sql.ErrNoRows,
			setErr:  func(m *WorkstationsMethods, e error) { m.wsStore = &fakeWSStore{deleteErr: e} },
			wantErr: protocol.ErrNotFound,
		},
		{
			name:    "delete legitimate",
			handler: func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleDelete },
			params:  map[string]any{"id": id},
			setErr:  func(m *WorkstationsMethods, e error) { m.wsStore = &fakeWSStore{deleteErr: e} },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewWorkstationsMethods(&fakeWSStore{}, &fakeLinkStore{})
			tc.setErr(m, tc.err)
			resp := callWSHandler(t, tc.handler(m), tc.params)
			if tc.wantErr == "" {
				if !resp.OK {
					t.Fatalf("response = %+v, want success", resp)
				}
				return
			}
			wantErrCode(t, resp, tc.wantErr)
		})
	}
}

// --- FIX E: mutations publish the lifecycle events ---

// TestWSMutationsPublishEvents is the FIX E regression: the events were
// subscribed to but never published, so the caches only expired at their TTL.
func TestWSMutationsPublishEvents(t *testing.T) {
	tests := []struct {
		name    string
		handler func(*WorkstationsMethods) gateway.MethodHandler
		params  any
		evType  eventbus.EventType
		setup   func(*WorkstationsMethods)
	}{
		{
			name:    "create publishes updated",
			handler: func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleCreate },
			params: map[string]any{
				"workstationKey": "ws-one",
				"name":           "one",
				"backendType":    store.BackendDocker,
				// The container backend reaches the runtime over SSH, so a
				// docker row needs the SSH transport too (store.ValidateMetadata
				// enforces the same shape the backend opens).
				"metadata": map[string]any{
					"image": "alpine", "host": "10.0.0.1", "user": "ubuntu", "privateKey": "PEM",
				},
			},
			evType: eventbus.EventWorkstationUpdated,
		},
		{
			name:    "update publishes updated",
			handler: func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleUpdate },
			params:  map[string]any{"id": uuid.NewString(), "updates": map[string]any{"name": "x"}},
			evType:  eventbus.EventWorkstationUpdated,
		},
		{
			name:    "delete publishes deleted",
			handler: func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleDelete },
			params:  map[string]any{"id": uuid.NewString()},
			evType:  eventbus.EventWorkstationDeleted,
		},
		{
			name:    "permission add publishes perm changed",
			handler: func(m *WorkstationsMethods) gateway.MethodHandler { return m.handlePermAdd },
			params:  map[string]any{"workstationId": uuid.NewString(), "pattern": "git"},
			evType:  eventbus.EventWorkstationPermChanged,
			setup: func(m *WorkstationsMethods) {
				m.permStore = &fakePermStore{}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := eventbus.NewDomainEventBus(eventbus.DefaultConfig())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			bus.Start(ctx)
			defer bus.Drain(time.Second) //nolint:errcheck

			got := make(chan eventbus.DomainEvent, 4)
			bus.Subscribe(tc.evType, func(_ context.Context, e eventbus.DomainEvent) error {
				got <- e
				return nil
			})

			m := NewWorkstationsMethods(&fakeWSStore{}, &fakeLinkStore{})
			m.SetDomainBus(bus)
			if tc.setup != nil {
				tc.setup(m)
			}
			resp := callWSHandler(t, tc.handler(m), tc.params)
			if !resp.OK {
				t.Fatalf("mutation failed: %+v", resp)
			}

			select {
			case e := <-got:
				p, ok := e.Payload.(eventbus.WorkstationChangedPayload)
				if !ok {
					t.Fatalf("payload type = %T, want WorkstationChangedPayload", e.Payload)
				}
				if _, err := uuid.Parse(p.WorkstationID); err != nil {
					t.Fatalf("payload workstation id %q is not a UUID: %v", p.WorkstationID, err)
				}
				if e.SourceID != "" {
					t.Fatalf("SourceID = %q; must be empty so the bus dedup cannot drop a repeat invalidation", e.SourceID)
				}
			case <-time.After(time.Second):
				t.Fatalf("no %s event published", tc.evType)
			}
		})
	}
}
