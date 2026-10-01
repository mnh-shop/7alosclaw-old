package pg

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/mnh-shop/7alosclaw/internal/store"
)

// seedWorkstation inserts a minimal workstation row for tenantID and registers
// no cleanup of its own: the tenant row is deleted by seedTenantAndAgent's
// cleanup and workstations cascade from it.
func seedWorkstation(t *testing.T, db *sql.DB, tenantID uuid.UUID, key string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	_, err := db.Exec(
		`INSERT INTO workstations
		 (id, workstation_key, tenant_id, name, backend_type, metadata, default_env)
		 VALUES ($1,$2,$3,$4,'docker',$5,$6)`,
		id, key, tenantID, "ws-"+key, []byte(`{}`), []byte(`{}`))
	if err != nil {
		t.Fatalf("seed workstation %s: %v", key, err)
	}
	return id
}

func linkCount(t *testing.T, db *sql.DB, agentID uuid.UUID, isDefault *bool) int {
	t.Helper()
	q := `SELECT count(*) FROM agent_workstation_links WHERE agent_id = $1`
	args := []any{agentID}
	if isDefault != nil {
		q += ` AND is_default = $2`
		args = append(args, *isDefault)
	}
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("count links: %v", err)
	}
	return n
}

// TestLinkRejectsCrossTenantEndpoints is the FIX A regression: Link stamped the
// caller's tenant onto the row without checking that the agent and workstation
// belong to that tenant, so a tenant-A admin could link tenant B's agent to A's
// workstation (and vice versa). Both directions must now fail closed.
func TestLinkRejectsCrossTenantEndpoints(t *testing.T) {
	db := hooksTestDB(t)
	tenantA, agentA := seedTenantAndAgent(t, db)
	tenantB, agentB := seedTenantAndAgent(t, db)
	wsA := seedWorkstation(t, db, tenantA, "ws-a")
	wsB := seedWorkstation(t, db, tenantB, "ws-b")

	s := NewPGAgentWorkstationLinkStore(db)
	ctxA := store.WithTenantID(context.Background(), tenantA)

	// B's agent onto A's workstation.
	err := s.Link(ctxA, &store.AgentWorkstationLink{AgentID: agentB, WorkstationID: wsA})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("agent B -> workstation A: err = %v, want sql.ErrNoRows", err)
	}
	// A's agent onto B's workstation.
	err = s.Link(ctxA, &store.AgentWorkstationLink{AgentID: agentA, WorkstationID: wsB})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("agent A -> workstation B: err = %v, want sql.ErrNoRows", err)
	}
	// Unknown ids are a not-found too.
	err = s.Link(ctxA, &store.AgentWorkstationLink{AgentID: uuid.Must(uuid.NewV7()), WorkstationID: wsA})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown agent -> workstation A: err = %v, want sql.ErrNoRows", err)
	}
	if n := linkCount(t, db, agentA, nil); n != 0 {
		t.Fatalf("cross-tenant attempts inserted %d link rows, want 0", n)
	}

	// A legitimate same-tenant link still succeeds.
	if err := s.Link(ctxA, &store.AgentWorkstationLink{AgentID: agentA, WorkstationID: wsA}); err != nil {
		t.Fatalf("same-tenant link: %v", err)
	}
	if n := linkCount(t, db, agentA, nil); n != 1 {
		t.Fatalf("same-tenant link count = %d, want 1", n)
	}
}

// TestLinkDefaultClearsPreviousDefault is the FIX B regression: Link inserted
// is_default directly, so a second default violated idx_agent_workstation_default
// and surfaced as a raw 500. The second default must now displace the first.
func TestLinkDefaultClearsPreviousDefault(t *testing.T) {
	db := hooksTestDB(t)
	tenantA, agentA := seedTenantAndAgent(t, db)
	ws1 := seedWorkstation(t, db, tenantA, "ws-default-1")
	ws2 := seedWorkstation(t, db, tenantA, "ws-default-2")

	s := NewPGAgentWorkstationLinkStore(db)
	ctx := store.WithTenantID(context.Background(), tenantA)

	if err := s.Link(ctx, &store.AgentWorkstationLink{AgentID: agentA, WorkstationID: ws1, IsDefault: true}); err != nil {
		t.Fatalf("first default link: %v", err)
	}
	if err := s.Link(ctx, &store.AgentWorkstationLink{AgentID: agentA, WorkstationID: ws2, IsDefault: true}); err != nil {
		t.Fatalf("second default link: %v", err)
	}

	defaults := true
	if n := linkCount(t, db, agentA, &defaults); n != 1 {
		t.Fatalf("default link count = %d, want exactly 1", n)
	}
	var defaultWS uuid.UUID
	if err := db.QueryRow(
		`SELECT workstation_id FROM agent_workstation_links WHERE agent_id = $1 AND is_default = TRUE`,
		agentA).Scan(&defaultWS); err != nil {
		t.Fatalf("read default: %v", err)
	}
	if defaultWS != ws2 {
		t.Fatalf("default workstation = %s, want the second one %s", defaultWS, ws2)
	}
}
