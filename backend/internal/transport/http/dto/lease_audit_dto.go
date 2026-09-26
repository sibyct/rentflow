package dto

import (
	"time"

	"propertymanagement/internal/domain"
)

// LeaseAuditEntryResponse mirrors AuditEntryResponse's shape for the
// separate lease_audit_log table (R18).
type LeaseAuditEntryResponse struct {
	ID        string                        `json:"id"`
	LeaseID   string                        `json:"lease_id"`
	Action    string                        `json:"action"`
	ActorID   string                        `json:"actor_id"`
	Changes   map[string]domain.FieldChange `json:"changes"`
	CreatedAt string                        `json:"created_at"`
}

func NewLeaseAuditListResponse(entries []*domain.LeaseAuditEntry) []LeaseAuditEntryResponse {
	out := make([]LeaseAuditEntryResponse, len(entries))
	for i, e := range entries {
		out[i] = LeaseAuditEntryResponse{
			ID:        e.ID.String(),
			LeaseID:   e.LeaseID.String(),
			Action:    e.Action,
			ActorID:   e.ActorID.String(),
			Changes:   e.Changes,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}
