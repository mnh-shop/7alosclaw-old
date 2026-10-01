package pg

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/store"
)

// TestWorkstationUpdateDeleteReportNoRows is the FIX C regression: Update and
// Delete ignored RowsAffected, so a nonexistent or cross-tenant id returned nil
// and the transports answered 200. They must now return sql.ErrNoRows.
func TestWorkstationUpdateDeleteReportNoRows(t *testing.T) {
	db := hooksTestDB(t)
	tenantA, _ := seedTenantAndAgent(t, db)
	tenantB, _ := seedTenantAndAgent(t, db)

	s := NewPGWorkstationStore(db, "")
	ctxA := store.WithTenantID(context.Background(), tenantA)
	ctxB := store.WithTenantID(context.Background(), tenantB)

	ws := &store.Workstation{
		WorkstationKey: "scope-ws",
		Name:           "scope ws",
		BackendType:    store.BackendDocker,
		Metadata:       []byte(`{}`),
		DefaultEnv:     []byte(`{}`),
		Active:         true,
	}
	if err := s.Create(ctxA, ws); err != nil {
		t.Fatalf("create: %v", err)
	}

	unknown := uuid.Must(uuid.NewV7())
	updates := map[string]any{"name": "renamed"}

	// Nonexistent id.
	if err := s.Update(ctxA, unknown, updates); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("update nonexistent: err = %v, want sql.ErrNoRows", err)
	}
	// Cross-tenant id.
	if err := s.Update(ctxB, ws.ID, updates); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-tenant update: err = %v, want sql.ErrNoRows", err)
	}
	// Legitimate update still succeeds.
	if err := s.Update(ctxA, ws.ID, updates); err != nil {
		t.Fatalf("same-tenant update: %v", err)
	}

	// Nonexistent delete.
	if err := s.Delete(ctxA, unknown); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("delete nonexistent: err = %v, want sql.ErrNoRows", err)
	}
	// Cross-tenant delete.
	if err := s.Delete(ctxB, ws.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-tenant delete: err = %v, want sql.ErrNoRows", err)
	}
	// Legitimate delete still succeeds and removes the row.
	if err := s.Delete(ctxA, ws.ID); err != nil {
		t.Fatalf("same-tenant delete: %v", err)
	}
	if _, err := s.GetByID(ctxA, ws.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("workstation still present after delete: %v", err)
	}
}
