import type { StatusTone } from '@/shared/components';

// Backed by the real staff-accounts API (backend/internal/domain/staff.go,
// service/staff_service.go, transport/http/handlers/staff_handler.go) —
// see api/staffApi.ts for the wire mapping.

export type StaffRole = 'admin' | 'property_manager' | 'maintenance_coordinator' | 'accountant';

export const STAFF_ROLES: StaffRole[] = ['admin', 'property_manager', 'maintenance_coordinator', 'accountant'];

export const ROLE_LABELS: Record<StaffRole, string> = {
  admin: 'Admin',
  property_manager: 'Property Manager',
  maintenance_coordinator: 'Maintenance Coordinator',
  accountant: 'Accountant',
};

export const ROLE_DESCRIPTIONS: Record<StaffRole, string> = {
  admin: 'Full access, including settings, users and billing.',
  property_manager: 'Leases, tenants, units and maintenance.',
  maintenance_coordinator: 'Work orders, vendors and scheduling.',
  accountant: 'Rent collection, payments and financial reports.',
};

export type UserStatus = 'active' | 'invited' | 'invite_expired' | 'deactivated';

export const USER_STATUS_LABELS: Record<UserStatus, string> = {
  active: 'Active',
  invited: 'Invited',
  invite_expired: 'Invite expired',
  deactivated: 'Deactivated',
};

export const USER_STATUS_TONE: Record<UserStatus, StatusTone> = {
  active: 'success',
  invited: 'info',
  invite_expired: 'warning',
  deactivated: 'default',
};

export interface PropertyAccess {
  all: boolean;
  /** Only meaningful when `all` is false — both arrays are the same order/length. */
  propertyIds: string[];
  propertyNames: string[];
}

export function allPropertyAccess(): PropertyAccess {
  return { all: true, propertyIds: [], propertyNames: [] };
}

export interface StaffUser {
  id: string;
  name: string;
  email: string;
  role: StaffRole;
  status: UserStatus;
  propertyAccess: PropertyAccess;
  /** ISO timestamp, or null for "Never" (an invite that hasn't been accepted). */
  lastLoginAt: string | null;
  /** Set only while status is 'invited' or 'invite_expired'. */
  inviteExpiresAt: string | null;
  /** True only for the signed-in user's own row ("You" tag) — cosmetic, not a permission. */
  isCurrentUser: boolean;
  /** The account's root owner — can't be edited, deactivated or reassigned through this API (see StaffService.requireManageableStaff). */
  isAccountOwner: boolean;
}

export type AuditEventKind = 'invite_sent' | 'invite_resent' | 'invite_accepted' | 'role_or_access_changed' | 'deactivated' | 'reactivated';

export const AUDIT_EVENT_LABELS: Record<AuditEventKind, string> = {
  invite_sent: 'Invite sent',
  invite_resent: 'Invite resent',
  invite_accepted: 'Invite accepted',
  role_or_access_changed: 'Role or access changed',
  deactivated: 'Deactivated',
  reactivated: 'Reactivated',
};

export interface AuditFieldChange {
  old: string;
  new: string;
}

export interface AuditEntry {
  id: string;
  when: string; // ISO timestamp
  event: AuditEventKind;
  subjectName: string;
  /** Keyed by field name ("role", "property_access"); empty for events with no before/after (invite sent, invite accepted). */
  changes: Record<string, AuditFieldChange>;
  changedBy: string;
}

/** Module × role → access level, shown in the Roles & Permissions matrix. Mirrors what RequireRole (backend/internal/transport/http/middleware/auth.go) would enforce if it were wired to any route today — it isn't; see the note under the matrix. */
export type PermissionLevel = 'full' | 'edit' | 'view' | 'none';

export interface PermissionRow {
  module: string;
  levels: Record<StaffRole, PermissionLevel>;
}

export const PERMISSION_MATRIX: PermissionRow[] = [
  { module: 'Properties & units', levels: { admin: 'full', property_manager: 'edit', maintenance_coordinator: 'view', accountant: 'view' } },
  { module: 'Leases & tenants', levels: { admin: 'full', property_manager: 'full', maintenance_coordinator: 'view', accountant: 'view' } },
  { module: 'Maintenance', levels: { admin: 'full', property_manager: 'full', maintenance_coordinator: 'full', accountant: 'none' } },
  { module: 'Vendors', levels: { admin: 'full', property_manager: 'view', maintenance_coordinator: 'full', accountant: 'view' } },
  { module: 'Accounting', levels: { admin: 'full', property_manager: 'view', maintenance_coordinator: 'none', accountant: 'full' } },
  { module: 'Reports', levels: { admin: 'full', property_manager: 'view', maintenance_coordinator: 'view', accountant: 'full' } },
  { module: 'Users & settings', levels: { admin: 'full', property_manager: 'none', maintenance_coordinator: 'none', accountant: 'none' } },
];
