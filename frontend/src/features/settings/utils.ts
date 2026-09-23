import type { PropertyAccess } from './types';

export function formatPropertyAccess(access: PropertyAccess): string {
  if (access.all) return 'All properties';
  if (access.properties.length === 0) return 'No properties';
  return `${access.properties.length} propert${access.properties.length === 1 ? 'y' : 'ies'}`;
}

/** The 1-2 property names shown under the count, e.g. "Willow Creek Apartments, Oak Terrace…". */
export function propertyAccessSubtext(access: PropertyAccess): string {
  if (access.all || access.properties.length === 0) return '';
  const shown = access.properties.slice(0, 2).join(', ');
  return access.properties.length > 2 ? `${shown}…` : shown;
}

/** "expires in 4 days" / "expired 2 days ago", for the Invited / Invite expired status subtext. */
export function inviteExpiryLabel(iso: string): string {
  const diffMs = new Date(iso).getTime() - Date.now();
  const diffDays = Math.round(Math.abs(diffMs) / 86_400_000);
  const unit = `day${diffDays === 1 ? '' : 's'}`;
  return diffMs >= 0 ? `expires in ${diffDays} ${unit}` : `expired ${diffDays} ${unit} ago`;
}

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  const first = parts[0]?.[0] ?? '';
  const last = parts.length > 1 ? (parts[parts.length - 1]?.[0] ?? '') : '';
  return (first + last).toUpperCase();
}
