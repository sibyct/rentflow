import { create } from 'zustand';
import { CURRENT_USER_ID, seedAuditLog, seedUsers } from '../mock/seedData';
import { formatPropertyAccess } from '../utils';
import {
  ROLE_LABELS,
  type AuditEntry,
  type AuditEventKind,
  type PropertyAccess,
  type StaffRole,
  type StaffUser,
} from '../types';

interface Toast {
  message: string;
  severity: 'success' | 'error';
}

export interface InviteInput {
  name: string;
  email: string;
  role: StaffRole;
  propertyAccess: PropertyAccess;
}

export type InviteResult = { kind: 'created' } | { kind: 'conflict'; existing: StaffUser };

interface UsersRolesState {
  users: StaffUser[];
  auditLog: AuditEntry[];
  toast: Toast | null;

  invite: (input: InviteInput) => InviteResult;
  resendInvite: (userId: string) => void;
  deactivate: (userId: string) => void;
  reactivate: (userId: string) => void;
  updateUser: (userId: string, changes: { role: StaffRole; propertyAccess: PropertyAccess }) => void;
  dismissToast: () => void;

  /** True while this user is the account's only active Admin — the safeguard the edit drawer and row menu both check. */
  isLastActiveAdmin: (userId: string) => boolean;
  activeAdminCount: () => number;
}

function newId(prefix: string) {
  return `${prefix}${Math.random().toString(36).slice(2, 9)}`;
}

function logEntry(event: AuditEventKind, subjectName: string, changedBy: string, from?: string, to?: string): AuditEntry {
  return { id: newId('a'), when: new Date().toISOString(), event, subjectName, from, to, changedBy };
}

export const useUsersRolesStore = create<UsersRolesState>((set, get) => ({
  users: seedUsers(),
  auditLog: seedAuditLog(),
  toast: null,

  activeAdminCount: () => get().users.filter((u) => u.role === 'admin' && u.status === 'active').length,

  isLastActiveAdmin: (userId) => {
    const user = get().users.find((u) => u.id === userId);
    if (!user || user.role !== 'admin' || user.status !== 'active') return false;
    return get().activeAdminCount() <= 1;
  },

  invite: (input) => {
    const email = input.email.trim().toLowerCase();
    const existing = get().users.find((u) => u.email.toLowerCase() === email);
    if (existing) return { kind: 'conflict', existing };

    const currentUser = get().users.find((u) => u.id === CURRENT_USER_ID);
    const user: StaffUser = {
      id: newId('u'),
      name: input.name.trim(),
      email: input.email.trim(),
      role: input.role,
      status: 'invited',
      propertyAccess: input.role === 'admin' ? { all: true, properties: [] } : input.propertyAccess,
      lastLoginAt: null,
      inviteExpiresAt: new Date(Date.now() + 7 * 86_400_000).toISOString(),
    };
    set((s) => ({
      users: [...s.users, user],
      auditLog: [logEntry('invite_sent', user.name, currentUser?.name ?? 'You'), ...s.auditLog],
      toast: { message: `Invitation sent to ${user.email}`, severity: 'success' },
    }));
    return { kind: 'created' };
  },

  resendInvite: (userId) => {
    const currentUser = get().users.find((u) => u.id === CURRENT_USER_ID);
    set((s) => ({
      users: s.users.map((u) =>
        u.id === userId
          ? { ...u, status: 'invited', inviteExpiresAt: new Date(Date.now() + 7 * 86_400_000).toISOString() }
          : u,
      ),
      auditLog: (() => {
        const u = s.users.find((x) => x.id === userId);
        return u ? [logEntry('invite_resent', u.name, currentUser?.name ?? 'You'), ...s.auditLog] : s.auditLog;
      })(),
      toast: (() => {
        const u = s.users.find((x) => x.id === userId);
        return u ? { message: `Invitation resent to ${u.email}`, severity: 'success' as const } : s.toast;
      })(),
    }));
  },

  deactivate: (userId) => {
    if (get().isLastActiveAdmin(userId)) {
      set({ toast: { message: 'Promote another user to Admin before deactivating the only active Admin.', severity: 'error' } });
      return;
    }
    const currentUser = get().users.find((u) => u.id === CURRENT_USER_ID);
    const user = get().users.find((u) => u.id === userId);
    if (!user) return;
    set((s) => ({
      users: s.users.map((u) => (u.id === userId ? { ...u, status: 'deactivated' } : u)),
      auditLog: [logEntry('deactivated', user.name, currentUser?.name ?? 'You'), ...s.auditLog],
      toast: { message: `${user.name} was deactivated`, severity: 'success' },
    }));
  },

  reactivate: (userId) => {
    const currentUser = get().users.find((u) => u.id === CURRENT_USER_ID);
    const user = get().users.find((u) => u.id === userId);
    if (!user) return;
    set((s) => ({
      users: s.users.map((u) => (u.id === userId ? { ...u, status: 'active' } : u)),
      auditLog: [logEntry('reactivated', user.name, currentUser?.name ?? 'You', 'Deactivated', 'Active'), ...s.auditLog],
      toast: { message: `${user.name} was reactivated`, severity: 'success' },
    }));
  },

  updateUser: (userId, changes) => {
    const currentUser = get().users.find((u) => u.id === CURRENT_USER_ID);
    const user = get().users.find((u) => u.id === userId);
    if (!user) return;
    if (user.role === 'admin' && user.status === 'active' && changes.role !== 'admin' && get().isLastActiveAdmin(userId)) {
      set({ toast: { message: 'Promote another user to Admin before changing the only active Admin’s role.', severity: 'error' } });
      return;
    }

    const entries: AuditEntry[] = [];
    if (changes.role !== user.role) {
      entries.push(logEntry('role_changed', user.name, currentUser?.name ?? 'You', ROLE_LABELS[user.role], ROLE_LABELS[changes.role]));
    }
    const nextAccess = changes.role === 'admin' ? { all: true, properties: [] } : changes.propertyAccess;
    const accessChanged = JSON.stringify(nextAccess) !== JSON.stringify(user.propertyAccess);
    if (accessChanged) {
      entries.push(
        logEntry(
          'property_access_changed',
          user.name,
          currentUser?.name ?? 'You',
          formatPropertyAccess(user.propertyAccess),
          formatPropertyAccess(nextAccess),
        ),
      );
    }

    set((s) => ({
      users: s.users.map((u) => (u.id === userId ? { ...u, role: changes.role, propertyAccess: nextAccess } : u)),
      auditLog: [...entries, ...s.auditLog],
      toast: entries.length > 0 ? { message: `${user.name}’s access was updated`, severity: 'success' } : s.toast,
    }));
  },

  dismissToast: () => set({ toast: null }),
}));
