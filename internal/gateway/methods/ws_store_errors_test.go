package methods

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/edition"
	"github.com/mnh-shop/7alosclaw/internal/gateway"
	"github.com/mnh-shop/7alosclaw/internal/hooks"
	"github.com/mnh-shop/7alosclaw/internal/permissions"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/pkg/protocol"
)

// Faulting stores for the WS error-classification tests (TASKS.md H64, same
// class as H61/H65). Each embeds the store interface and overrides only the
// methods the handlers under test call, returning a configurable error. A
// wrapped sql.ErrNoRows must answer NOT_FOUND with a generic message; any
// other store failure must answer INTERNAL without echoing the driver's
// text. A code-only assertion would pass against the unfixed code, which
// answered INVALID_REQUEST/INTERNAL to every failure and echoed err.Error().

type faultingWorkstationStore struct {
	store.WorkstationStore
	getErr    error
	listErr   error
	createErr error
	updateErr error
	deleteErr error
}

func (s *faultingWorkstationStore) GetByID(context.Context, uuid.UUID) (*store.Workstation, error) {
	return nil, s.getErr
}
func (s *faultingWorkstationStore) List(context.Context) ([]store.Workstation, error) {
	return nil, s.listErr
}
func (s *faultingWorkstationStore) Create(context.Context, *store.Workstation) error {
	return s.createErr
}
func (s *faultingWorkstationStore) Update(context.Context, uuid.UUID, map[string]any) error {
	return s.updateErr
}
func (s *faultingWorkstationStore) Delete(context.Context, uuid.UUID) error { return s.deleteErr }

type faultingWSLinkStore struct {
	store.AgentWorkstationLinkStore
	linkErr   error
	unlinkErr error
}

func (s *faultingWSLinkStore) Link(context.Context, *store.AgentWorkstationLink) error {
	return s.linkErr
}
func (s *faultingWSLinkStore) Unlink(context.Context, uuid.UUID, uuid.UUID) error {
	return s.unlinkErr
}

type faultingTeamStore struct {
	store.TeamStore
	getErr         error
	membersErr     error
	deleteErr      error
	updateErr      error
	tasksErr       error
	activeTasksErr error
	knownUsersErr  error
	scopesErr      error
	eventsErr      error
}

func (s *faultingTeamStore) GetTeam(context.Context, uuid.UUID) (*store.TeamData, error) {
	return nil, s.getErr
}
func (s *faultingTeamStore) ListMembers(context.Context, uuid.UUID) ([]store.TeamMemberData, error) {
	return nil, s.membersErr
}
func (s *faultingTeamStore) DeleteTeam(context.Context, uuid.UUID) error { return s.deleteErr }
func (s *faultingTeamStore) UpdateTeam(context.Context, uuid.UUID, map[string]any) error {
	return s.updateErr
}
func (s *faultingTeamStore) ListTasks(context.Context, uuid.UUID, string, string, string, string, string, int, int) ([]store.TeamTaskData, error) {
	return nil, s.tasksErr
}
func (s *faultingTeamStore) ListActiveTasksByChatID(context.Context, string) ([]store.TeamTaskData, error) {
	return nil, s.activeTasksErr
}
func (s *faultingTeamStore) KnownUserIDs(context.Context, uuid.UUID, int) ([]string, error) {
	return nil, s.knownUsersErr
}
func (s *faultingTeamStore) ListTaskScopes(context.Context, uuid.UUID) ([]store.ScopeEntry, error) {
	return nil, s.scopesErr
}
func (s *faultingTeamStore) ListTeamEvents(context.Context, uuid.UUID, int, int) ([]store.TeamTaskEventData, error) {
	return nil, s.eventsErr
}

type faultingHookStore struct {
	hooks.HookStore
	getErr    error
	listErr   error
	createErr error
	updateErr error
	deleteErr error
}

// GetByID returns a valid tenant-scoped config when getErr is unset, so the
// update path can reach the store.Update fault through the merged-config
// re-validation.
func (s *faultingHookStore) GetByID(context.Context, uuid.UUID) (*hooks.HookConfig, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return &hooks.HookConfig{
		Event:       hooks.EventPreToolUse,
		Scope:       hooks.ScopeTenant,
		HandlerType: hooks.HandlerHTTP,
		TenantID:    uuid.MustParse(hookStoreErrorsTenantID),
		Config:      map[string]any{"url": "https://example.com/hook"},
	}, nil
}
func (s *faultingHookStore) List(context.Context, hooks.ListFilter) ([]hooks.HookConfig, error) {
	return nil, s.listErr
}
func (s *faultingHookStore) Create(context.Context, hooks.HookConfig) (uuid.UUID, error) {
	return uuid.Nil, s.createErr
}
func (s *faultingHookStore) Update(context.Context, uuid.UUID, map[string]any) error {
	return s.updateErr
}
func (s *faultingHookStore) Delete(context.Context, uuid.UUID) error { return s.deleteErr }

const (
	wsStoreErrorsID    = "00000000-0000-0000-0000-0000000000b1"
	agentStoreErrorsID = "00000000-0000-0000-0000-0000000000b2"
	// Non-sentinel tenant for the hook fixtures: Validate rejects tenant-scoped
	// hooks whose TenantID is the sentinel UUID unless the caller is master scope.
	hookStoreErrorsTenantID = "0193a5b0-7000-7000-8000-00000000000a"
)

// assertClassifiedError runs the handler and pins the classification: the
// response code must match, and the message must not leak the driver's text.
func assertClassifiedError(t *testing.T, ctx context.Context, handle gateway.MethodHandler, params map[string]any, wantCode string) {
	t.Helper()
	client, out := newOwnerClient(t)
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	req := &protocol.RequestFrame{ID: "req", Method: "test", Params: raw}

	resp := callAndDecode(t, ctx, client, out, handle, req)

	if resp.OK || resp.Error == nil {
		t.Fatalf("expected an error frame, got %#v", resp)
	}
	if resp.Error.Code != wantCode {
		t.Fatalf("code = %q, want %q (message %q)", resp.Error.Code, wantCode, resp.Error.Message)
	}
	for _, leak := range []string{"relation", "does not exist", "driver", "pq:"} {
		if strings.Contains(resp.Error.Message, leak) {
			t.Fatalf("message %q leaked driver text %q", resp.Error.Message, leak)
		}
	}
}

// TestWorkstationsWSErrorClassificationDoesNotLeakDriverText pins the
// workstations.* store-fault classification per handler.
func TestWorkstationsWSErrorClassificationDoesNotLeakDriverText(t *testing.T) {
	notFound := sql.ErrNoRows
	driverFault := errors.New(`pq: relation "workstations" does not exist`)
	ctx := context.Background()
	const wsID = wsStoreErrorsID
	const agentID = agentStoreErrorsID

	cases := []struct {
		name     string
		setup    func(*faultingWorkstationStore, *faultingWSLinkStore)
		handle   func(*WorkstationsMethods) gateway.MethodHandler
		params   map[string]any
		wantCode string
	}{
		{
			name:     "get maps an absent row to NOT_FOUND",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { w.getErr = notFound },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleGet },
			params:   map[string]any{"id": wsID},
			wantCode: protocol.ErrNotFound,
		},
		{
			name:     "get maps a database fault to INTERNAL without the driver text",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { w.getErr = driverFault },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleGet },
			params:   map[string]any{"id": wsID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "create maps a database fault to INTERNAL without the driver text",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { w.createErr = driverFault },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleCreate },
			params:   map[string]any{"workstationKey": "ws-a", "backendType": "ssh", "metadata": map[string]any{"host": "h", "user": "u", "privateKey": "pem"}},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "update maps an absent row to NOT_FOUND",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { w.updateErr = notFound },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleUpdate },
			params:   map[string]any{"id": wsID, "updates": map[string]any{"name": "Renamed"}},
			wantCode: protocol.ErrNotFound,
		},
		{
			name:     "update maps a database fault to INTERNAL without the driver text",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { w.updateErr = driverFault },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleUpdate },
			params:   map[string]any{"id": wsID, "updates": map[string]any{"name": "Renamed"}},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "update metadata lookup maps a database fault to INTERNAL without the driver text",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { w.getErr = driverFault },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleUpdate },
			params:   map[string]any{"id": wsID, "updates": map[string]any{"metadata": map[string]any{"host": "h2"}}},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "delete maps a database fault to INTERNAL without the driver text",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { w.deleteErr = driverFault },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleDelete },
			params:   map[string]any{"id": wsID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "test connection lookup maps a database fault to INTERNAL without the driver text",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { w.getErr = driverFault },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleTestConnection },
			params:   map[string]any{"id": wsID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "link agent maps a database fault to INTERNAL without the driver text",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { l.linkErr = driverFault },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleLinkAgent },
			params:   map[string]any{"agentId": agentID, "workstationId": wsID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "unlink agent maps a database fault to INTERNAL without the driver text",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { l.unlinkErr = driverFault },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleUnlinkAgent },
			params:   map[string]any{"agentId": agentID, "workstationId": wsID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "unlink agent maps an absent row to NOT_FOUND",
			setup:    func(w *faultingWorkstationStore, l *faultingWSLinkStore) { l.unlinkErr = notFound },
			handle:   func(m *WorkstationsMethods) gateway.MethodHandler { return m.handleUnlinkAgent },
			params:   map[string]any{"agentId": agentID, "workstationId": wsID},
			wantCode: protocol.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wsStore, linkStore := &faultingWorkstationStore{}, &faultingWSLinkStore{}
			tc.setup(wsStore, linkStore)
			m := NewWorkstationsMethods(wsStore, linkStore)
			assertClassifiedError(t, ctx, tc.handle(m), tc.params, tc.wantCode)
		})
	}
}

// TestTeamsWSErrorClassificationDoesNotLeakDriverText pins the teams.* (CRUD
// and dashboard read) store-fault classification per handler.
func TestTeamsWSErrorClassificationDoesNotLeakDriverText(t *testing.T) {
	notFound := sql.ErrNoRows
	driverFault := errors.New(`pq: relation "teams" does not exist`)
	ctx := context.Background()
	const teamID = wsStoreErrorsID

	cases := []struct {
		name     string
		setup    func(*faultingTeamStore)
		handle   func(*TeamsMethods) gateway.MethodHandler
		params   map[string]any
		wantCode string
	}{
		{
			name:     "get maps an absent row to NOT_FOUND",
			setup:    func(s *faultingTeamStore) { s.getErr = notFound },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleGet },
			params:   map[string]any{"teamId": teamID},
			wantCode: protocol.ErrNotFound,
		},
		{
			name:     "get maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.getErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleGet },
			params:   map[string]any{"teamId": teamID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "get members maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.membersErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleGet },
			params:   map[string]any{"teamId": teamID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "delete maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.deleteErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleDelete },
			params:   map[string]any{"teamId": teamID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "update lookup maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.getErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleUpdate },
			params:   map[string]any{"teamId": teamID, "name": "Renamed"},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "update write maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.updateErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleUpdate },
			params:   map[string]any{"teamId": teamID, "name": "Renamed"},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "task list maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.tasksErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleTaskList },
			params:   map[string]any{"teamId": teamID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "active tasks by session maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.activeTasksErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleTaskActiveBySession },
			params:   map[string]any{"sessionKey": "sess-a"},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "known users maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.knownUsersErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleKnownUsers },
			params:   map[string]any{"teamId": teamID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "scopes maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.scopesErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleScopes },
			params:   map[string]any{"teamId": teamID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "events list maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingTeamStore) { s.eventsErr = driverFault },
			handle:   func(m *TeamsMethods) gateway.MethodHandler { return m.handleEventsList },
			params:   map[string]any{"team_id": teamID},
			wantCode: protocol.ErrInternal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &faultingTeamStore{}
			tc.setup(fake)
			m := NewTeamsMethods(fake, nil, nil, nil, nil, nil, "")
			assertClassifiedError(t, ctx, tc.handle(m), tc.params, tc.wantCode)
		})
	}
}

// TestHooksWSErrorClassificationDoesNotLeakDriverText pins the hooks.*
// store-fault classification per handler. Validation errors keep their
// actionable text and are not covered here.
func TestHooksWSErrorClassificationDoesNotLeakDriverText(t *testing.T) {
	notFound := sql.ErrNoRows
	driverFault := errors.New(`pq: relation "hooks" does not exist`)
	const tenantID = hookStoreErrorsTenantID
	ctx := store.WithTenantID(context.Background(), uuid.MustParse(tenantID))
	const hookID = wsStoreErrorsID

	cases := []struct {
		name     string
		setup    func(*faultingHookStore)
		handle   func(*HookMethods) gateway.MethodHandler
		params   map[string]any
		wantCode string
	}{
		{
			name:     "list maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingHookStore) { s.listErr = driverFault },
			handle:   func(m *HookMethods) gateway.MethodHandler { return m.handleList },
			params:   map[string]any{},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "create maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingHookStore) { s.createErr = driverFault },
			handle:   func(m *HookMethods) gateway.MethodHandler { return m.handleCreate },
			params:   map[string]any{"event": "pre_tool_use", "scope": "tenant", "handler_type": "http", "config": map[string]any{"url": "https://example.com/hook"}},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "update lookup maps an absent row to NOT_FOUND",
			setup:    func(s *faultingHookStore) { s.getErr = notFound },
			handle:   func(m *HookMethods) gateway.MethodHandler { return m.handleUpdate },
			params:   map[string]any{"hookId": hookID, "updates": map[string]any{"name": "Renamed"}},
			wantCode: protocol.ErrNotFound,
		},
		{
			name:     "update lookup maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingHookStore) { s.getErr = driverFault },
			handle:   func(m *HookMethods) gateway.MethodHandler { return m.handleUpdate },
			params:   map[string]any{"hookId": hookID, "updates": map[string]any{"name": "Renamed"}},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "update write maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingHookStore) { s.updateErr = driverFault },
			handle:   func(m *HookMethods) gateway.MethodHandler { return m.handleUpdate },
			params:   map[string]any{"hookId": hookID, "updates": map[string]any{"name": "Renamed"}},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "delete maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingHookStore) { s.deleteErr = driverFault },
			handle:   func(m *HookMethods) gateway.MethodHandler { return m.handleDelete },
			params:   map[string]any{"hookId": hookID},
			wantCode: protocol.ErrInternal,
		},
		{
			name:     "toggle maps a database fault to INTERNAL without the driver text",
			setup:    func(s *faultingHookStore) { s.updateErr = driverFault },
			handle:   func(m *HookMethods) gateway.MethodHandler { return m.handleToggle },
			params:   map[string]any{"hookId": hookID, "enabled": false},
			wantCode: protocol.ErrInternal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &faultingHookStore{}
			tc.setup(fake)
			m := NewHookMethods(fake, edition.Lite)
			assertClassifiedError(t, ctx, tc.handle(m), tc.params, tc.wantCode)
		})
	}
}

// newOwnerClient returns a capturing WS test client acting as the owner.
func newOwnerClient(t *testing.T) (*gateway.Client, <-chan []byte) {
	t.Helper()
	return gateway.NewCapturingTestClient(permissions.RoleOwner, uuid.Nil, "owner", 4)
}

// callAndDecode runs the handler against the capturing client and decodes the
// single response frame it emits, failing on silence.
func callAndDecode(t *testing.T, ctx context.Context, client *gateway.Client, out <-chan []byte,
	h gateway.MethodHandler, req *protocol.RequestFrame) *protocol.ResponseFrame {
	t.Helper()
	h(ctx, client, req)
	select {
	case raw := <-out:
		var resp protocol.ResponseFrame
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		return &resp
	case <-time.After(time.Second):
		t.Fatal("handler sent no response")
		return nil
	}
}
