package base

import (
	"fmt"

	"github.com/google/uuid"
)

// TenantIDForInsert returns tid if non-nil, otherwise fallback.
// Callers extract tenant ID from context before calling this.
//
// PERMISSIVE BY DESIGN. Use it only where a missing tenant is legitimately
// master-scoped: startup seeding, backup/restore, and other system operations.
//
// For a write reached from a REQUEST, use RequireTenantID instead. A request that
// lost its tenant context must fail closed rather than write into the platform's
// shared bucket. The predicate for "is this legitimately master" is
// store.IsExplicitMasterScope — NOT store.IsMasterScope, which treats an unset
// tenant as master and would defeat the check.
func TenantIDForInsert(tid, fallback uuid.UUID) uuid.UUID {
	if tid == uuid.Nil {
		return fallback
	}
	return tid
}

// RequireTenantID returns an error if tid is nil (fail-closed).
// Callers extract tenant ID from context before calling this.
func RequireTenantID(tid uuid.UUID) error {
	if tid == uuid.Nil {
		return fmt.Errorf("tenant_id required")
	}
	return nil
}
