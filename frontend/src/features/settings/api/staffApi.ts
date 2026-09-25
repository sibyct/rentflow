import { apiClient } from '@/api/client';
import type { AuditEntry, AuditEventKind, AuditFieldChange, PropertyAccess, StaffRole, StaffUser, UserStatus } from '../types';

// Wire shapes exactly as internal/transport/http/dto/staff_dto.go serializes them.

interface PropertyAccessWire {
  all: boolean;
  property_ids?: string[];
  property_names?: string[];
}

interface StaffMemberWire {
  id: string;
  name: string;
  email: string;
  role: string;
  status: string;
  property_access: PropertyAccessWire;
  is_account_owner: boolean;
  is_current_user: boolean;
  invited_by_name?: string;
  invite_expires_at?: string;
  last_login_at?: string;
  created_at: string;
}

interface InviteResultWire {
  member?: StaffMemberWire;
  conflict?: StaffMemberWire;
  invite_url?: string;
}

interface InviteLookupWire {
  name: string;
  email: string;
  role: string;
  property_access: PropertyAccessWire;
  invited_by_name: string;
  expired: boolean;
}

interface StaffAuditEntryWire {
  id: string;
  action: string;
  actor_name: string;
  subject_name: string;
  changes: Record<string, { old: string; new: string }>;
  created_at: string;
}

interface AuditListMetaWire {
  total: number;
  limit: number;
  offset: number;
}

interface PasswordResetLookupWire {
  email: string;
  expired: boolean;
}

function toPropertyAccess(wire: PropertyAccessWire): PropertyAccess {
  return { all: wire.all, propertyIds: wire.property_ids ?? [], propertyNames: wire.property_names ?? [] };
}

/** The write-side shape (property ids only, no names to resolve) — distinct from PropertyAccess, the read shape the API decorates with display names. */
export interface PropertyAccessInput {
  all: boolean;
  propertyIds: string[];
}

function toPropertyAccessRequest(access: PropertyAccessInput) {
  return access.all ? { all: true, property_ids: [] } : { all: false, property_ids: access.propertyIds };
}

function toStaffUser(wire: StaffMemberWire): StaffUser {
  return {
    id: wire.id,
    name: wire.name,
    email: wire.email,
    role: wire.role as StaffRole,
    status: wire.status as UserStatus,
    propertyAccess: toPropertyAccess(wire.property_access),
    lastLoginAt: wire.last_login_at ?? null,
    inviteExpiresAt: wire.invite_expires_at ?? null,
    isCurrentUser: wire.is_current_user,
    isAccountOwner: wire.is_account_owner,
  };
}

function toAuditEntry(wire: StaffAuditEntryWire): AuditEntry {
  const changes: Record<string, AuditFieldChange> = {};
  for (const [field, change] of Object.entries(wire.changes ?? {})) {
    changes[field] = { old: change.old, new: change.new };
  }
  return {
    id: wire.id,
    when: wire.created_at,
    event: wire.action as AuditEventKind,
    subjectName: wire.subject_name,
    changes,
    changedBy: wire.actor_name,
  };
}

export interface InviteInput {
  name: string;
  email: string;
  role: StaffRole;
  propertyAccess: PropertyAccessInput;
}

export type InviteResult = { kind: 'created'; user: StaffUser; inviteUrl: string } | { kind: 'conflict'; existing: StaffUser };

export interface UpdateStaffInput {
  role: StaffRole;
  propertyAccess: PropertyAccessInput;
}

export interface InviteLookup {
  name: string;
  email: string;
  role: StaffRole;
  propertyAccess: PropertyAccess;
  invitedByName: string;
  expired: boolean;
}

export interface AuditLogResult {
  entries: AuditEntry[];
  total: number;
}

export interface PasswordResetLookup {
  email: string;
  expired: boolean;
}

export const staffApi = {
  list: (): Promise<StaffUser[]> => apiClient.get<StaffMemberWire[]>('/api/v1/staff').then((rows) => rows.map(toStaffUser)),

  invite: (input: InviteInput): Promise<InviteResult> =>
    apiClient
      .post<InviteResultWire>('/api/v1/staff/invite', {
        name: input.name.trim(),
        email: input.email.trim(),
        role: input.role,
        property_access: toPropertyAccessRequest(input.propertyAccess),
      })
      .then((wire) => {
        if (wire.conflict) return { kind: 'conflict', existing: toStaffUser(wire.conflict) };
        // Created is always set when Conflict isn't — see domain.StaffInviteResult.
        return { kind: 'created', user: toStaffUser(wire.member!), inviteUrl: wire.invite_url ?? '' };
      }),

  update: (userId: string, input: UpdateStaffInput): Promise<StaffUser> =>
    apiClient
      .put<StaffMemberWire>(`/api/v1/staff/${userId}`, { role: input.role, property_access: toPropertyAccessRequest(input.propertyAccess) })
      .then(toStaffUser),

  deactivate: (userId: string): Promise<void> => apiClient.post<void>(`/api/v1/staff/${userId}/deactivate`),

  reactivate: (userId: string): Promise<void> => apiClient.post<void>(`/api/v1/staff/${userId}/reactivate`),

  resendInvite: (userId: string): Promise<{ inviteUrl: string }> =>
    apiClient.post<{ invite_url: string }>(`/api/v1/staff/${userId}/resend-invite`).then((r) => ({ inviteUrl: r.invite_url })),

  auditLog: (limit: number, offset: number): Promise<AuditLogResult> =>
    apiClient
      .getWithMeta<StaffAuditEntryWire[], AuditListMetaWire>(`/api/v1/staff/audit?limit=${limit}&offset=${offset}`)
      .then(({ data, meta }) => ({ entries: data.map(toAuditEntry), total: meta.total })),

  lookupInvite: (token: string): Promise<InviteLookup> =>
    apiClient.get<InviteLookupWire>(`/api/v1/invites?token=${encodeURIComponent(token)}`).then((wire) => ({
      name: wire.name,
      email: wire.email,
      role: wire.role as StaffRole,
      propertyAccess: toPropertyAccess(wire.property_access),
      invitedByName: wire.invited_by_name,
      expired: wire.expired,
    })),

  acceptInvite: (token: string, password: string): Promise<void> =>
    apiClient.post<void>('/api/v1/invites/accept', { token, password }),

  /** Admin-triggered: emails userId a reset link, never touches their status/role. */
  resetPassword: (userId: string): Promise<{ resetUrl: string }> =>
    apiClient.post<{ reset_url: string }>(`/api/v1/staff/${userId}/reset-password`).then((r) => ({ resetUrl: r.reset_url })),

  lookupPasswordReset: (token: string): Promise<PasswordResetLookup> =>
    apiClient.get<PasswordResetLookupWire>(`/api/v1/password-reset?token=${encodeURIComponent(token)}`),

  confirmPasswordReset: (token: string, password: string): Promise<void> =>
    apiClient.post<void>('/api/v1/password-reset/confirm', { token, password }),
};
