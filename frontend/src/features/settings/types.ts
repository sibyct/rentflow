import type { StatusTone } from '@/shared/components';

// This whole feature is a front-end prototype: RentFlow's backend has no
// staff/roles concept yet (an account is exactly one user — see
// backend/internal/domain/user.go). Everything here lives in the
// in-memory useUsersRolesStore and resets on reload; nothing is
// persisted or sent to the API. Kept in its own feature folder so it's
// obvious where the real thing would slot in once a backend exists.

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

/** The portfolio a property-access checklist picks from — mirrors the shape of a real property list without depending on the properties feature. */
export const MOCK_PROPERTIES = [
  'Willow Creek Apartments',
  'Oak Terrace',
  'Harbor View',
  'Cedar Point',
  'Riverside Commons',
  'Maple Grove',
];

export interface PropertyAccess {
  all: boolean;
  /** Only meaningful when `all` is false. */
  properties: string[];
}

export function allPropertyAccess(): PropertyAccess {
  return { all: true, properties: [] };
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
  /** True only for the signed-in demo account ("You" tag) — cosmetic, not a permission. */
  isCurrentUser?: boolean;
}

export type AuditEventKind =
  | 'invite_sent'
  | 'invite_resent'
  | 'role_changed'
  | 'property_access_changed'
  | 'deactivated'
  | 'reactivated';

export const AUDIT_EVENT_LABELS: Record<AuditEventKind, string> = {
  invite_sent: 'Invite sent',
  invite_resent: 'Invite resent',
  role_changed: 'Role changed',
  property_access_changed: 'Property access changed',
  deactivated: 'Deactivated',
  reactivated: 'Reactivated',
};

export interface AuditEntry {
  id: string;
  when: string; // ISO timestamp
  event: AuditEventKind;
  subjectName: string;
  /** Human-readable before/after, e.g. role names or "2 properties" → "3 properties". Omitted for events with no before state (invite sent). */
  from?: string;
  to?: string;
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
