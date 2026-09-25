package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

type PropertyAccessRequest struct {
	All         bool     `json:"all"`
	PropertyIDs []string `json:"property_ids" validate:"max=200,dive,uuid4"`
}

func (r PropertyAccessRequest) toDomain() (domain.PropertyAccess, error) {
	if r.All {
		return domain.AllPropertyAccess(), nil
	}
	ids := make([]uuid.UUID, 0, len(r.PropertyIDs))
	for _, s := range r.PropertyIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return domain.PropertyAccess{}, fmt.Errorf("property_ids: %w", err)
		}
		ids = append(ids, id)
	}
	return domain.PropertyAccess{All: false, PropertyIDs: ids}, nil
}

type InviteStaffRequest struct {
	Name           string                `json:"name" validate:"required,min=1,max=120,noctrl"`
	Email          string                `json:"email" validate:"required,email"`
	Role           string                `json:"role" validate:"required,oneof=admin property_manager maintenance_coordinator accountant"`
	PropertyAccess PropertyAccessRequest `json:"property_access"`
}

func (r *InviteStaffRequest) Sanitize() {
	r.Name = sanitizeString(r.Name)
	r.Email = sanitizeEmail(r.Email)
}

func (r *InviteStaffRequest) ToDomain() (domain.InviteStaffInput, error) {
	access, err := r.PropertyAccess.toDomain()
	if err != nil {
		return domain.InviteStaffInput{}, err
	}
	return domain.InviteStaffInput{Name: r.Name, Email: r.Email, Role: domain.StaffRole(r.Role), PropertyAccess: access}, nil
}

type UpdateStaffRequest struct {
	Role           string                `json:"role" validate:"required,oneof=admin property_manager maintenance_coordinator accountant"`
	PropertyAccess PropertyAccessRequest `json:"property_access"`
}

func (r *UpdateStaffRequest) ToDomain() (domain.UpdateStaffInput, error) {
	access, err := r.PropertyAccess.toDomain()
	if err != nil {
		return domain.UpdateStaffInput{}, err
	}
	return domain.UpdateStaffInput{Role: domain.StaffRole(r.Role), PropertyAccess: access}, nil
}

type AcceptInviteRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type ConfirmPasswordResetRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// PropertyAccessResponse mirrors PropertyAccessRequest for output, with
// display names alongside the ids so the client never has to cross-
// reference a separate properties call just to render "3 properties".
type PropertyAccessResponse struct {
	All           bool     `json:"all"`
	PropertyIDs   []string `json:"property_ids,omitempty"`
	PropertyNames []string `json:"property_names,omitempty"`
}

func newPropertyAccessResponse(access domain.PropertyAccess, names []string) PropertyAccessResponse {
	if access.All {
		return PropertyAccessResponse{All: true}
	}
	ids := make([]string, len(access.PropertyIDs))
	for i, id := range access.PropertyIDs {
		ids[i] = id.String()
	}
	return PropertyAccessResponse{All: false, PropertyIDs: ids, PropertyNames: names}
}

type StaffMemberResponse struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Email           string                 `json:"email"`
	Role            string                 `json:"role"`
	Status          string                 `json:"status"`
	PropertyAccess  PropertyAccessResponse `json:"property_access"`
	IsAccountOwner  bool                   `json:"is_account_owner"`
	IsCurrentUser   bool                   `json:"is_current_user"`
	InvitedByName   string                 `json:"invited_by_name,omitempty"`
	InviteExpiresAt string                 `json:"invite_expires_at,omitempty"`
	LastLoginAt     string                 `json:"last_login_at,omitempty"`
	CreatedAt       string                 `json:"created_at"`
}

func NewStaffMemberResponse(m *domain.StaffMember, now time.Time, currentActorID uuid.UUID) StaffMemberResponse {
	resp := StaffMemberResponse{
		ID: m.ID.String(), Name: m.Name, Email: m.Email, Role: string(m.EffectiveRole()),
		Status: string(m.DisplayStatus(now)), PropertyAccess: newPropertyAccessResponse(m.PropertyAccess, m.PropertyNames),
		IsAccountOwner: m.IsAccountOwner, IsCurrentUser: m.ID == currentActorID,
		InvitedByName: m.InvitedByName, CreatedAt: m.CreatedAt.Format(time.RFC3339),
	}
	if m.InviteExpiresAt != nil {
		resp.InviteExpiresAt = m.InviteExpiresAt.Format(time.RFC3339)
	}
	if m.LastLoginAt != nil {
		resp.LastLoginAt = m.LastLoginAt.Format(time.RFC3339)
	}
	return resp
}

func NewStaffMemberListResponse(members []*domain.StaffMember, now time.Time, currentActorID uuid.UUID) []StaffMemberResponse {
	out := make([]StaffMemberResponse, len(members))
	for i, m := range members {
		out[i] = NewStaffMemberResponse(m, now, currentActorID)
	}
	return out
}

// InviteResultResponse is what POST /staff/invite returns either way —
// exactly one of member/conflict is set, mirroring domain.StaffInviteResult.
type InviteResultResponse struct {
	Member    *StaffMemberResponse `json:"member,omitempty"`
	Conflict  *StaffMemberResponse `json:"conflict,omitempty"`
	InviteURL string               `json:"invite_url,omitempty"`
}

type InviteLookupResponse struct {
	Name           string                 `json:"name"`
	Email          string                 `json:"email"`
	Role           string                 `json:"role"`
	PropertyAccess PropertyAccessResponse `json:"property_access"`
	InvitedByName  string                 `json:"invited_by_name"`
	Expired        bool                   `json:"expired"`
}

func NewInviteLookupResponse(l *domain.InviteLookup) InviteLookupResponse {
	return InviteLookupResponse{
		Name: l.Name, Email: l.Email, Role: string(l.Role),
		PropertyAccess: newPropertyAccessResponse(l.PropertyAccess, l.PropertyNames),
		InvitedByName:  l.InvitedByName, Expired: l.Expired,
	}
}

type PasswordResetLookupResponse struct {
	Email   string `json:"email"`
	Expired bool   `json:"expired"`
}

func NewPasswordResetLookupResponse(l *domain.PasswordResetLookup) PasswordResetLookupResponse {
	return PasswordResetLookupResponse{Email: l.Email, Expired: l.Expired}
}

type StaffAuditEntryResponse struct {
	ID          string                        `json:"id"`
	Action      string                        `json:"action"`
	ActorName   string                        `json:"actor_name"`
	SubjectName string                        `json:"subject_name"`
	Changes     map[string]domain.FieldChange `json:"changes"`
	CreatedAt   string                        `json:"created_at"`
}

func NewStaffAuditListResponse(entries []*domain.StaffAuditEntry) []StaffAuditEntryResponse {
	out := make([]StaffAuditEntryResponse, len(entries))
	for i, e := range entries {
		out[i] = StaffAuditEntryResponse{
			ID: e.ID.String(), Action: e.Action, ActorName: e.ActorName, SubjectName: e.SubjectName,
			Changes: e.Changes, CreatedAt: e.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}
