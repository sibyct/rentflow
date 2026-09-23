// Display types for the Maintenance feature — work orders and the
// recurring rules that generate them. Mirrors features/leases/types.ts's
// role: no mock data, this is real from day one.

export type WorkOrderCategory = 'plumbing' | 'electrical' | 'hvac' | 'appliance' | 'pest_control' | 'general' | 'other';

export const WORK_ORDER_CATEGORIES: WorkOrderCategory[] = ['plumbing', 'electrical', 'hvac', 'appliance', 'pest_control', 'general', 'other'];

export const WORK_ORDER_CATEGORY_LABELS: Record<WorkOrderCategory, string> = {
  plumbing: 'Plumbing',
  electrical: 'Electrical',
  hvac: 'HVAC',
  appliance: 'Appliance',
  pest_control: 'Pest control',
  general: 'General',
  other: 'Other',
};

export type WorkOrderPriority = 'low' | 'medium' | 'high' | 'emergency';

export const WORK_ORDER_PRIORITIES: WorkOrderPriority[] = ['low', 'medium', 'high', 'emergency'];

export const WORK_ORDER_PRIORITY_LABELS: Record<WorkOrderPriority, string> = {
  low: 'Low',
  medium: 'Medium',
  high: 'High',
  emergency: 'Emergency',
};

// Matches the backend's EmergencySLAHours (domain package) — the
// window, from now, an emergency work order's due date must fall
// within. Duplicated here only for the form's own upper-bound hint;
// the backend is the actual source of truth and re-validates this on
// create.
export const EMERGENCY_SLA_HOURS = 24;

export type WorkOrderStatus = 'new' | 'assigned' | 'in_progress' | 'on_hold' | 'completed' | 'cancelled';

export const WORK_ORDER_STATUSES: WorkOrderStatus[] = ['new', 'assigned', 'in_progress', 'on_hold', 'completed', 'cancelled'];

export const WORK_ORDER_STATUS_LABELS: Record<WorkOrderStatus, string> = {
  new: 'New',
  assigned: 'Assigned',
  in_progress: 'In Progress',
  on_hold: 'On Hold',
  completed: 'Completed',
  cancelled: 'Cancelled',
};

export interface WorkOrderRow {
  id: string;
  propertyId: string;
  propertyName: string;
  unitId: string;
  unitName: string;
  title: string;
  category: WorkOrderCategory;
  priority: WorkOrderPriority;
  status: WorkOrderStatus;
  isOverdue: boolean;
  assignedTo: string;
  /** Set only when assignedTo came from a real Vendor record rather than freeform text — see WorkOrder.VendorID's backend doc comment. */
  vendorId: string;
  vendorName: string;
  /** 1-5 star rating a manager gives the vendor after completion; null until rated. */
  rating: number | null;
  dueDate: string;
  createdAt: string;
}

/** Full record for the work order drawer — everything WorkOrderRow has, plus every field the drawer's form collects. */
export interface WorkOrderDetail extends WorkOrderRow {
  description: string;
  reportedBy: string;
  reportedByContact: string;
  assignedToContact: string;
  accessInstructions: string;
  scheduledStart: string;
  scheduledEnd: string;
  estimatedCost: number | null;
  actualCost: number | null;
  /** Attachment id, or '' for none — uploaded via shared/components/FileUpload. */
  photoAttachmentId: string;
  invoiceAttachmentId: string;
  internalNotes: string;
  recurringRuleId: string;
  completedAt: string;
  updatedAt: string;
}

export type MaintenanceActivityKind = 'note' | 'status_change';
export type MaintenanceVisibility = 'internal' | 'tenant_visible';

export interface MaintenanceActivityEntry {
  id: string;
  workOrderId: string;
  kind: MaintenanceActivityKind;
  visibility: MaintenanceVisibility;
  message: string;
  oldValue: string;
  newValue: string;
  createdAt: string;
}

/** MaintenanceActivityEntry decorated with its work order/property — for the Dashboard's portfolio-wide activity feed, where (unlike the per-work-order activity panel) the work order isn't already known from the page. */
export interface MaintenanceActivityWithContext extends MaintenanceActivityEntry {
  workOrderTitle: string;
  propertyName: string;
}

export interface WorkOrderSummary {
  open: number;
  overdue: number;
  unassigned: number;
  emergency: number;
  high: number;
  medium: number;
  low: number;
}

export type RecurringRuleFrequencyUnit = 'days' | 'weeks' | 'months';

export const RECURRING_RULE_FREQUENCY_UNITS: RecurringRuleFrequencyUnit[] = ['days', 'weeks', 'months'];

export const RECURRING_RULE_FREQUENCY_UNIT_LABELS: Record<RecurringRuleFrequencyUnit, string> = {
  days: 'day(s)',
  weeks: 'week(s)',
  months: 'month(s)',
};

export interface RecurringRuleRow {
  id: string;
  propertyId: string;
  propertyName: string;
  unitId: string;
  unitName: string;
  title: string;
  description: string;
  category: WorkOrderCategory;
  frequencyInterval: number;
  frequencyUnit: RecurringRuleFrequencyUnit;
  nextDueDate: string;
  active: boolean;
}
