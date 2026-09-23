import { allPropertyAccess, type AuditEntry, type StaffUser } from '../types';

const hoursAgo = (h: number) => new Date(Date.now() - h * 3600_000).toISOString();
const daysAgo = (d: number) => hoursAgo(d * 24);

export const CURRENT_USER_ID = 'u1';

export function seedUsers(): StaffUser[] {
  return [
    {
      id: 'u1',
      name: 'Alex Doran',
      email: 'alex@harborpg.com',
      role: 'admin',
      status: 'active',
      propertyAccess: allPropertyAccess(),
      lastLoginAt: hoursAgo(0),
      inviteExpiresAt: null,
      isCurrentUser: true,
    },
    {
      id: 'u2',
      name: 'Priya Shah',
      email: 'priya.shah@harborpg.com',
      role: 'admin',
      status: 'active',
      propertyAccess: allPropertyAccess(),
      lastLoginAt: hoursAgo(2),
      inviteExpiresAt: null,
    },
    {
      id: 'u3',
      name: 'Marcus Bell',
      email: 'marcus.bell@harborpg.com',
      role: 'property_manager',
      status: 'active',
      propertyAccess: { all: false, properties: ['Willow Creek Apartments', 'Oak Terrace', 'Cedar Point'] },
      lastLoginAt: daysAgo(1),
      inviteExpiresAt: null,
    },
    {
      id: 'u4',
      name: 'Dana Whitfield',
      email: 'dana.w@harborpg.com',
      role: 'property_manager',
      status: 'active',
      propertyAccess: { all: false, properties: ['Oak Terrace', 'Harbor View'] },
      lastLoginAt: daysAgo(3),
      inviteExpiresAt: null,
    },
    {
      id: 'u5',
      name: 'Luis Ortega',
      email: 'luis.ortega@harborpg.com',
      role: 'maintenance_coordinator',
      status: 'active',
      propertyAccess: allPropertyAccess(),
      lastLoginAt: hoursAgo(6),
      inviteExpiresAt: null,
    },
    {
      id: 'u6',
      name: 'Sofia Reyes',
      email: 'sofia.reyes@harborpg.com',
      role: 'maintenance_coordinator',
      status: 'active',
      propertyAccess: { all: false, properties: ['Riverside Commons', 'Maple Grove', 'Cedar Point', 'Harbor View'] },
      lastLoginAt: daysAgo(5),
      inviteExpiresAt: null,
    },
    {
      id: 'u7',
      name: 'Grace Kim',
      email: 'grace.kim@harborpg.com',
      role: 'accountant',
      status: 'active',
      propertyAccess: allPropertyAccess(),
      lastLoginAt: daysAgo(7),
      inviteExpiresAt: null,
    },
    {
      id: 'u8',
      name: 'Tom Reyes',
      email: 'tom.reyes@harborpg.com',
      role: 'accountant',
      status: 'invited',
      propertyAccess: allPropertyAccess(),
      lastLoginAt: null,
      inviteExpiresAt: daysAgo(-4), // 4 days from now
    },
    {
      id: 'u9',
      name: 'Owen Park',
      email: 'owen.park@harborpg.com',
      role: 'property_manager',
      status: 'invite_expired',
      propertyAccess: { all: false, properties: ['Maple Grove'] },
      lastLoginAt: null,
      inviteExpiresAt: daysAgo(2), // expired 2 days ago
    },
    {
      id: 'u10',
      name: 'Ben Carter',
      email: 'ben.carter@harborpg.com',
      role: 'maintenance_coordinator',
      status: 'deactivated',
      propertyAccess: { all: false, properties: ['Willow Creek Apartments', 'Oak Terrace'] },
      lastLoginAt: daysAgo(24),
      inviteExpiresAt: null,
    },
  ];
}

export function seedAuditLog(): AuditEntry[] {
  const entries: AuditEntry[] = [
    { id: 'a1', when: daysAgo(2.7), event: 'role_changed', subjectName: 'Dana Whitfield', from: 'Maintenance Coordinator', to: 'Property Manager', changedBy: 'Alex Doran' },
    { id: 'a2', when: daysAgo(6.3), event: 'invite_sent', subjectName: 'Tom Reyes', changedBy: 'Alex Doran' },
    { id: 'a3', when: daysAgo(20.4), event: 'deactivated', subjectName: 'Ben Carter', changedBy: 'Alex Doran' },
    { id: 'a4', when: daysAgo(25.2), event: 'property_access_changed', subjectName: 'Marcus Bell', from: '2 properties', to: '3 properties', changedBy: 'Priya Shah' },
    { id: 'a5', when: daysAgo(39), event: 'reactivated', subjectName: 'Luis Ortega', from: 'Deactivated', to: 'Active', changedBy: 'Alex Doran' },
    { id: 'a6', when: daysAgo(45), event: 'invite_sent', subjectName: 'Owen Park', changedBy: 'Priya Shah' },
  ];
  return entries.sort((x, y) => (x.when < y.when ? 1 : -1));
}
