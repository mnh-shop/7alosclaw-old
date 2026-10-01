package backup

import (
	"strings"
	"testing"
)

// TestWorkstationFamilyRegistered is the FIX D regression: the four workstation
// tables were absent from TenantTables(), so a tenant backup silently omitted
// every workstation, link and permission. The three durable tables must now be
// registered; workstation_activity is an ephemeral audit log and is skipped
// alongside activity_logs.
func TestWorkstationFamilyRegistered(t *testing.T) {
	lookup := make(map[string]TableDef)
	for _, tbl := range TenantTables() {
		lookup[tbl.Name] = tbl
	}

	// workstations is a root tenant table (Tier 2); the two child tables are
	// Tier 3 and carry tenant_id directly.
	if tbl, ok := lookup["workstations"]; !ok {
		t.Error("workstations missing from backup registry")
	} else if tbl.Tier != 2 || !tbl.HasTenantID {
		t.Errorf("workstations = tier %d, HasTenantID %v; want tier 2, true", tbl.Tier, tbl.HasTenantID)
	}

	links, ok := lookup["agent_workstation_links"]
	if !ok {
		t.Error("agent_workstation_links missing from backup registry")
	} else {
		if links.Tier != 3 || !links.HasTenantID {
			t.Errorf("agent_workstation_links = tier %d, HasTenantID %v; want tier 3, true", links.Tier, links.HasTenantID)
		}
		// PK is (agent_id, workstation_id) — no id column, so ORDER BY id fails.
		q, err := links.exportQuery()
		if err != nil {
			t.Fatalf("agent_workstation_links exportQuery: %v", err)
		}
		if strings.Contains(q, "ORDER BY id") {
			t.Errorf("agent_workstation_links export orders by missing id column: %q", q)
		}
		if !strings.Contains(q, "ORDER BY agent_id, workstation_id") {
			t.Errorf("agent_workstation_links export must order by its composite PK: %q", q)
		}
	}

	if tbl, ok := lookup["workstation_permissions"]; !ok {
		t.Error("workstation_permissions missing from backup registry")
	} else if tbl.Tier != 3 || !tbl.HasTenantID {
		t.Errorf("workstation_permissions = tier %d, HasTenantID %v; want tier 3, true", tbl.Tier, tbl.HasTenantID)
	}

	// workstation_activity is a rolling audit log pruned nightly: registered
	// would replay per-run history into a restored tenant, so it follows the
	// activity_logs precedent and is skipped explicitly.
	if _, ok := lookup["workstation_activity"]; ok {
		t.Error("workstation_activity must not be exported; it belongs in skippedTenantTables")
	}
	if !skippedTenantTables["workstation_activity"] {
		t.Error("workstation_activity missing from skippedTenantTables")
	}
}
