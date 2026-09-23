import { apiClient } from '@/api/client';
import type {
  MaintenanceActivityEntry,
  MaintenanceActivityWithContext,
  MaintenanceVisibility,
  WorkOrderCategory,
  WorkOrderDetail,
  WorkOrderPriority,
  WorkOrderRow,
  WorkOrderStatus,
  WorkOrderSummary,
} from '../types';
import type { WorkOrderFormValues } from '../schemas/workOrderSchema';

// Wire shape exactly as internal/transport/http/dto/work_order_dto.go's
// WorkOrderWithPropertyResponse serializes it.
interface WorkOrderWire {
  id: string;
  property_id: string;
  property_name: string;
  unit_id?: string;
  unit_name?: string;
  title: string;
  description?: string;
  category: string;
  priority: string;
  status: string;
  is_overdue: boolean;
  reported_by?: string;
  reported_by_contact?: string;
  assigned_to?: string;
  assigned_to_contact?: string;
  vendor_id?: string;
  vendor_name?: string;
  rating?: number | null;
  access_instructions?: string;
  scheduled_start?: string;
  scheduled_end?: string;
  due_date?: string;
  estimated_cost?: number | null;
  actual_cost?: number | null;
  photo_attachment_id?: string;
  invoice_attachment_id?: string;
  internal_notes?: string;
  recurring_rule_id?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

interface WorkOrderListMetaWire {
  total: number;
  limit: number;
  offset: number;
}

interface MaintenanceActivityWire {
  id: string;
  work_order_id: string;
  kind: string;
  visibility: string;
  message: string;
  old_value?: string;
  new_value?: string;
  created_at: string;
}

interface WorkOrderSummaryWire {
  open: number;
  overdue: number;
  unassigned: number;
  emergency: number;
  high: number;
  medium: number;
  low: number;
}

interface MaintenanceActivityWithContextWire extends MaintenanceActivityWire {
  work_order_title: string;
  property_name: string;
}

function toWorkOrderDetail(wire: WorkOrderWire): WorkOrderDetail {
  return {
    id: wire.id,
    propertyId: wire.property_id,
    propertyName: wire.property_name,
    unitId: wire.unit_id ?? '',
    unitName: wire.unit_name ?? '',
    title: wire.title,
    category: wire.category as WorkOrderCategory,
    priority: wire.priority as WorkOrderPriority,
    status: wire.status as WorkOrderStatus,
    isOverdue: wire.is_overdue,
    assignedTo: wire.assigned_to ?? '',
    vendorId: wire.vendor_id ?? '',
    vendorName: wire.vendor_name ?? '',
    rating: wire.rating ?? null,
    dueDate: wire.due_date ?? '',
    createdAt: wire.created_at,
    description: wire.description ?? '',
    reportedBy: wire.reported_by ?? '',
    reportedByContact: wire.reported_by_contact ?? '',
    assignedToContact: wire.assigned_to_contact ?? '',
    accessInstructions: wire.access_instructions ?? '',
    scheduledStart: wire.scheduled_start ?? '',
    scheduledEnd: wire.scheduled_end ?? '',
    estimatedCost: wire.estimated_cost ?? null,
    actualCost: wire.actual_cost ?? null,
    photoAttachmentId: wire.photo_attachment_id ?? '',
    invoiceAttachmentId: wire.invoice_attachment_id ?? '',
    internalNotes: wire.internal_notes ?? '',
    recurringRuleId: wire.recurring_rule_id ?? '',
    completedAt: wire.completed_at ?? '',
    updatedAt: wire.updated_at,
  };
}

// WorkOrderRow is a strict subset of WorkOrderDetail's fields, so every
// list row is just its detail projection — same pattern as units/leases.
function toWorkOrderRow(wire: WorkOrderWire): WorkOrderRow {
  return toWorkOrderDetail(wire);
}

function toMaintenanceActivity(wire: MaintenanceActivityWire): MaintenanceActivityEntry {
  return {
    id: wire.id,
    workOrderId: wire.work_order_id,
    kind: wire.kind as MaintenanceActivityEntry['kind'],
    visibility: wire.visibility as MaintenanceVisibility,
    message: wire.message,
    oldValue: wire.old_value ?? '',
    newValue: wire.new_value ?? '',
    createdAt: wire.created_at,
  };
}

function toMaintenanceActivityWithContext(wire: MaintenanceActivityWithContextWire): MaintenanceActivityWithContext {
  return {
    ...toMaintenanceActivity(wire),
    workOrderTitle: wire.work_order_title,
    propertyName: wire.property_name,
  };
}

/** Prefills the work order drawer's form from an existing record — the reverse of toWorkOrderRequest below. */
export function toFormValues(detail: WorkOrderDetail): WorkOrderFormValues {
  return {
    unitId: detail.unitId,
    title: detail.title,
    description: detail.description,
    category: detail.category,
    priority: detail.priority,
    status: detail.status,
    reportedBy: detail.reportedBy,
    reportedByContact: detail.reportedByContact,
    assignedTo: detail.assignedTo,
    assignedToContact: detail.assignedToContact,
    vendorId: detail.vendorId,
    rating: detail.rating != null ? String(detail.rating) : '',
    accessInstructions: detail.accessInstructions,
    scheduledStart: toLocalDateTimeInput(detail.scheduledStart),
    scheduledEnd: toLocalDateTimeInput(detail.scheduledEnd),
    dueDate: detail.dueDate,
    estimatedCost: detail.estimatedCost != null ? String(detail.estimatedCost) : '',
    actualCost: detail.actualCost != null ? String(detail.actualCost) : '',
    photoAttachmentId: detail.photoAttachmentId,
    invoiceAttachmentId: detail.invoiceAttachmentId,
    internalNotes: detail.internalNotes,
  };
}

/** RFC3339 -> the value a `datetime-local` input expects ("YYYY-MM-DDTHH:mm"). */
function toLocalDateTimeInput(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function parseNumber(value: string | undefined): number | undefined {
  if (!value || value.trim() === '') return undefined;
  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
}

/** "YYYY-MM-DDTHH:mm" (datetime-local) -> RFC3339, or undefined if empty/invalid. */
function toISOOrUndefined(localValue: string | undefined): string | undefined {
  if (!localValue) return undefined;
  const d = new Date(localValue);
  return Number.isNaN(d.getTime()) ? undefined : d.toISOString();
}

function toWorkOrderRequest(values: WorkOrderFormValues) {
  return {
    unit_id: values.unitId || undefined,
    title: values.title.trim(),
    description: values.description?.trim() || undefined,
    category: values.category,
    priority: values.priority || undefined,
    status: values.status || undefined,
    reported_by: values.reportedBy?.trim() || undefined,
    reported_by_contact: values.reportedByContact?.trim() || undefined,
    assigned_to: values.assignedTo?.trim() || undefined,
    assigned_to_contact: values.assignedToContact?.trim() || undefined,
    vendor_id: values.vendorId || undefined,
    rating: parseNumber(values.rating),
    access_instructions: values.accessInstructions?.trim() || undefined,
    scheduled_start: toISOOrUndefined(values.scheduledStart),
    scheduled_end: toISOOrUndefined(values.scheduledEnd),
    due_date: values.dueDate || undefined,
    estimated_cost: parseNumber(values.estimatedCost),
    actual_cost: parseNumber(values.actualCost),
    // Always sent, even when empty: the backend's update contract reads
    // an empty string as "clear it" and a missing field as "leave
    // unchanged" (see UpdateWorkOrderRequest.PhotoAttachmentID) — since
    // this form is always fully populated from the current record (see
    // toFormValues), there's never a "leave unchanged" case to omit for.
    photo_attachment_id: values.photoAttachmentId?.trim() ?? '',
    invoice_attachment_id: values.invoiceAttachmentId?.trim() ?? '',
    internal_notes: values.internalNotes?.trim() || undefined,
  };
}

export type WorkOrderSortKey = 'priority' | 'dueDate' | 'createdAt' | 'status';
const SORT_TO_WIRE: Record<WorkOrderSortKey, string> = {
  priority: 'priority',
  dueDate: 'due_date',
  createdAt: 'created_at',
  status: 'status',
};

export interface WorkOrderListParams {
  search?: string;
  propertyId?: string;
  status?: WorkOrderStatus | '';
  priority?: WorkOrderPriority | '';
  category?: WorkOrderCategory | '';
  assignedTo?: string;
  vendorId?: string;
  overdue?: boolean;
  unassigned?: boolean;
  sort?: WorkOrderSortKey;
  order?: 'asc' | 'desc';
  limit: number;
  offset: number;
}

export interface WorkOrderListResult {
  workOrders: WorkOrderRow[];
  total: number;
}

function buildListQuery(params: WorkOrderListParams): string {
  const q = new URLSearchParams();
  if (params.search) q.set('search', params.search);
  if (params.propertyId) q.set('property_id', params.propertyId);
  if (params.status) q.set('status', params.status);
  if (params.priority) q.set('priority', params.priority);
  if (params.category) q.set('category', params.category);
  if (params.assignedTo) q.set('assigned_to', params.assignedTo);
  if (params.vendorId) q.set('vendor_id', params.vendorId);
  if (params.overdue) q.set('overdue', 'true');
  if (params.unassigned) q.set('unassigned', 'true');
  if (params.sort) q.set('sort', SORT_TO_WIRE[params.sort]);
  if (params.order) q.set('order', params.order);
  q.set('limit', String(params.limit));
  q.set('offset', String(params.offset));
  return q.toString();
}

export const workOrdersApi = {
  list: (params: WorkOrderListParams): Promise<WorkOrderListResult> =>
    apiClient
      .getWithMeta<WorkOrderWire[], WorkOrderListMetaWire>(`/api/v1/work-orders?${buildListQuery(params)}`)
      .then(({ data, meta }) => ({ workOrders: data.map(toWorkOrderRow), total: meta.total })),

  get: (id: string): Promise<WorkOrderDetail> =>
    apiClient.get<WorkOrderWire>(`/api/v1/work-orders/${id}`).then(toWorkOrderDetail),

  getSummary: (): Promise<WorkOrderSummary> =>
    apiClient.get<WorkOrderSummaryWire>('/api/v1/work-orders/summary').then((s) => ({
      open: s.open,
      overdue: s.overdue,
      unassigned: s.unassigned,
      emergency: s.emergency,
      high: s.high,
      medium: s.medium,
      low: s.low,
    })),

  /** propertyId comes from context (the property/unit page the flow was opened from), never a form field. */
  create: (propertyId: string, values: WorkOrderFormValues): Promise<WorkOrderDetail> =>
    apiClient
      .post<{ id: string }>(`/api/v1/properties/${propertyId}/work-orders`, toWorkOrderRequest(values))
      .then((r) => workOrdersApi.get(r.id)),

  update: (id: string, values: WorkOrderFormValues): Promise<WorkOrderDetail> =>
    apiClient.put<{ id: string }>(`/api/v1/work-orders/${id}`, toWorkOrderRequest(values)).then((r) => workOrdersApi.get(r.id)),

  delete: (id: string): Promise<void> => apiClient.delete<void>(`/api/v1/work-orders/${id}`),

  listActivity: (workOrderId: string): Promise<MaintenanceActivityEntry[]> =>
    apiClient.get<MaintenanceActivityWire[]>(`/api/v1/work-orders/${workOrderId}/activity`).then((rows) => rows.map(toMaintenanceActivity)),

  /** Backs the Dashboard's activity feed — the most recent activity across every work order the caller owns, not scoped to one. */
  getRecentActivity: (limit: number, offset: number): Promise<MaintenanceActivityWithContext[]> =>
    apiClient
      .get<MaintenanceActivityWithContextWire[]>(`/api/v1/work-orders/activity?limit=${limit}&offset=${offset}`)
      .then((rows) => rows.map(toMaintenanceActivityWithContext)),

  addNote: (workOrderId: string, message: string, visibility: MaintenanceVisibility): Promise<MaintenanceActivityEntry> =>
    apiClient
      .post<MaintenanceActivityWire>(`/api/v1/work-orders/${workOrderId}/activity`, { message, visibility })
      .then(toMaintenanceActivity),

  bulkUpdateStatus: (ids: string[], status: WorkOrderStatus): Promise<number> =>
    apiClient.patch<{ updated: number }>('/api/v1/work-orders/status', { ids, status }).then((r) => r.updated),

  bulkReassign: (ids: string[], assignedTo: string): Promise<number> =>
    apiClient.patch<{ updated: number }>('/api/v1/work-orders/reassign', { ids, assigned_to: assignedTo }).then((r) => r.updated),
};
