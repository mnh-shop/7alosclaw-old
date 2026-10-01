package workstation

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/eventbus"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/internal/workstation/security"
)

// PublishWorkstationEvent emits a workstation lifecycle event so the caches
// derived from the workstation row (BackendCache, AllowlistChecker) are
// invalidated at once instead of at TTL expiry. bus may be nil
// tests) — the call is then a no-op.
//
// The workstation UUID is carried in the payload, not SourceID: the bus dedups
// on SourceID for five minutes, which would drop a second edit of the same
// workstation and leave a stale cache entry until its TTL (10m for the backend,
// 30s for the allowlist).
func PublishWorkstationEvent(ctx context.Context, bus eventbus.DomainEventBus, evType eventbus.EventType, wsID uuid.UUID) {
	if bus == nil || wsID == uuid.Nil {
		return
	}
	var tenantID string
	if tid := store.TenantIDFromContext(ctx); tid != uuid.Nil {
		tenantID = tid.String()
	}
	bus.Publish(eventbus.DomainEvent{
		ID:        store.GenNewID().String(),
		Type:      evType,
		TenantID:  tenantID,
		Timestamp: time.Now(),
		Payload:   eventbus.WorkstationChangedPayload{WorkstationID: wsID.String()},
	})
}

// SubscribeWorkstationInvalidations wires the workstation caches to the
// lifecycle events. It returns an unsubscribe function.
//
// checker may be nil (the permission store is not wired); the permission
// subscription is then skipped. cache must be non-nil.
func SubscribeWorkstationInvalidations(
	bus eventbus.DomainEventBus,
	cache *BackendCache,
	checker *security.AllowlistChecker,
) func() {
	if bus == nil {
		return func() {}
	}

	var unsubs []func()
	if cache != nil {
		unsubs = append(unsubs,
			bus.Subscribe(eventbus.EventWorkstationUpdated, func(_ context.Context, e eventbus.DomainEvent) error {
				if id, ok := workstationIDFromEvent(e); ok {
					cache.Invalidate(id)
				}
				return nil
			}),
			bus.Subscribe(eventbus.EventWorkstationDeleted, func(_ context.Context, e eventbus.DomainEvent) error {
				if id, ok := workstationIDFromEvent(e); ok {
					cache.Invalidate(id)
				}
				return nil
			}),
		)
	}
	if checker != nil {
		unsubs = append(unsubs,
			bus.Subscribe(eventbus.EventWorkstationPermChanged, func(_ context.Context, e eventbus.DomainEvent) error {
				if id, ok := workstationIDFromEvent(e); ok {
					checker.Invalidate(id)
				}
				return nil
			}),
		)
	}

	return func() {
		for _, u := range unsubs {
			u()
		}
	}
}

// workstationIDFromEvent reads the workstation UUID from the typed payload,
// falling back to SourceID for publishers that still put it there.
func workstationIDFromEvent(e eventbus.DomainEvent) (uuid.UUID, bool) {
	if p, ok := e.Payload.(eventbus.WorkstationChangedPayload); ok {
		if id, err := uuid.Parse(p.WorkstationID); err == nil {
			return id, true
		}
	}
	if id, err := uuid.Parse(e.SourceID); err == nil {
		return id, true
	}
	return uuid.Nil, false
}
