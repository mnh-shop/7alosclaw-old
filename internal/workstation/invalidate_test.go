package workstation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/eventbus"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/internal/workstation/security"
)

type fakePermStore struct {
	patterns []string
}

func (f *fakePermStore) ListForWorkstation(context.Context, uuid.UUID) ([]store.WorkstationPermission, error) {
	perms := make([]store.WorkstationPermission, 0, len(f.patterns))
	for _, p := range f.patterns {
		perms = append(perms, store.WorkstationPermission{Pattern: p, Enabled: true})
	}
	return perms, nil
}
func (f *fakePermStore) GetByID(context.Context, uuid.UUID) (*store.WorkstationPermission, error) {
	return nil, nil
}
func (f *fakePermStore) Add(context.Context, *store.WorkstationPermission) error { return nil }
func (f *fakePermStore) Remove(context.Context, uuid.UUID) error                 { return nil }
func (f *fakePermStore) SetEnabled(context.Context, uuid.UUID, bool) error       { return nil }
func (f *fakePermStore) SeedDefaults(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

// TestPermissionEventInvalidatesAllowlist is the FIX E regression: the
// workstation.perm.changed subscription had no publisher, so a revoked
// permission stayed effective for the 30s allowlist TTL. A long TTL here means
// the test can only pass if the invalidation actually runs.
func TestPermissionEventInvalidatesAllowlist(t *testing.T) {
	perms := &fakePermStore{patterns: []string{"git"}}
	checker := security.NewAllowlistChecker(perms, time.Hour)

	bus := eventbus.NewDomainEventBus(eventbus.DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)
	defer bus.Drain(time.Second) //nolint:errcheck

	unsub := SubscribeWorkstationInvalidations(bus, nil, checker)
	defer unsub()

	ws := &store.Workstation{ID: uuid.New(), TenantID: uuid.New()}

	// Prime the cache: git is allowed.
	if err := checker.Check(ctx, ws, "git", nil); err != nil {
		t.Fatalf("git should be allowed before revocation: %v", err)
	}
	// Revoke git by replacing the allowlist, then publish the invalidation.
	perms.patterns = []string{"ls"}
	PublishWorkstationEvent(ctx, bus, eventbus.EventWorkstationPermChanged, ws.ID)

	waitFor(t, func() bool {
		return checker.Check(ctx, ws, "git", nil) != nil
	}, "revoked permission was never invalidated")
}

type countingWSStore struct {
	workstation *store.Workstation
	gets        int
}

func (s *countingWSStore) Create(context.Context, *store.Workstation) error { return nil }
func (s *countingWSStore) GetByID(context.Context, uuid.UUID) (*store.Workstation, error) {
	s.gets++
	return s.workstation, nil
}
func (s *countingWSStore) GetByKey(context.Context, string) (*store.Workstation, error) {
	return s.workstation, nil
}
func (s *countingWSStore) List(context.Context) ([]store.Workstation, error) { return nil, nil }
func (s *countingWSStore) Update(context.Context, uuid.UUID, map[string]any) error {
	return nil
}
func (s *countingWSStore) SetActive(context.Context, uuid.UUID, bool) error { return nil }
func (s *countingWSStore) Delete(context.Context, uuid.UUID) error          { return nil }

type noopBackend struct{}

func (noopBackend) Name() string                                         { return "test-backend" }
func (noopBackend) HealthCheck(context.Context) error                    { return nil }
func (noopBackend) OpenSession(context.Context, string) (Session, error) { return nil, nil }
func (noopBackend) CloseSession(context.Context, string) error           { return nil }
func (noopBackend) Close() error                                         { return nil }

// TestUpdateEventInvalidatesBackendCache pins the other half of FIX E: an edit
// or delete must evict the cached backend immediately, not after the 10m TTL.
func TestUpdateEventInvalidatesBackendCache(t *testing.T) {
	const backendType = store.WorkstationBackend("test-invalidate-backend")
	Register(backendType, func(*store.Workstation) (Backend, error) { return noopBackend{}, nil })

	wsID := uuid.New()
	wsStore := &countingWSStore{workstation: &store.Workstation{
		ID:          wsID,
		BackendType: backendType,
		Active:      true,
	}}
	cache := NewBackendCache(wsStore, time.Hour)

	bus := eventbus.NewDomainEventBus(eventbus.DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)
	defer bus.Drain(time.Second) //nolint:errcheck

	unsub := SubscribeWorkstationInvalidations(bus, cache, nil)
	defer unsub()

	if _, err := cache.Get(ctx, wsID); err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if _, err := cache.Get(ctx, wsID); err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if wsStore.gets != 1 {
		t.Fatalf("cache miss count = %d after two Gets, want 1 (second was a hit)", wsStore.gets)
	}

	PublishWorkstationEvent(ctx, bus, eventbus.EventWorkstationUpdated, wsID)
	waitFor(t, func() bool {
		if _, err := cache.Get(ctx, wsID); err != nil {
			t.Fatalf("Get after invalidation: %v", err)
		}
		return wsStore.gets > 1
	}, "backend cache was never invalidated")
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(msg)
}
