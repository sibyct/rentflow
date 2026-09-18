import { apiClient } from '@/api/client';
import type { RecurringRuleFrequencyUnit, RecurringRuleRow, WorkOrderCategory } from '../types';
import type { RecurringRuleFormValues } from '../schemas/recurringRuleSchema';

// Wire shape exactly as internal/transport/http/dto/maintenance_rule_dto.go's
// RecurringRuleWithPropertyResponse serializes it.
interface RecurringRuleWire {
  id: string;
  property_id: string;
  property_name: string;
  unit_id?: string;
  unit_name?: string;
  title: string;
  description?: string;
  category: string;
  frequency_interval: number;
  frequency_unit: string;
  next_due_date: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

function toRecurringRuleRow(wire: RecurringRuleWire): RecurringRuleRow {
  return {
    id: wire.id,
    propertyId: wire.property_id,
    propertyName: wire.property_name,
    unitId: wire.unit_id ?? '',
    unitName: wire.unit_name ?? '',
    title: wire.title,
    description: wire.description ?? '',
    category: wire.category as WorkOrderCategory,
    frequencyInterval: wire.frequency_interval,
    frequencyUnit: wire.frequency_unit as RecurringRuleFrequencyUnit,
    nextDueDate: wire.next_due_date,
    active: wire.active,
  };
}

export function toFormValues(rule: RecurringRuleRow): RecurringRuleFormValues {
  return {
    unitId: rule.unitId,
    title: rule.title,
    description: rule.description,
    category: rule.category,
    frequencyInterval: String(rule.frequencyInterval),
    frequencyUnit: rule.frequencyUnit,
    nextDueDate: rule.nextDueDate,
  };
}

function toCreateRecurringRuleRequest(values: RecurringRuleFormValues) {
  return {
    unit_id: values.unitId || undefined,
    title: values.title.trim(),
    description: values.description?.trim() || undefined,
    category: values.category,
    frequency_interval: Number(values.frequencyInterval),
    frequency_unit: values.frequencyUnit,
    next_due_date: values.nextDueDate,
  };
}

export const recurringRulesApi = {
  list: (): Promise<RecurringRuleRow[]> =>
    apiClient.get<RecurringRuleWire[]>('/api/v1/maintenance-rules').then((rows) => rows.map(toRecurringRuleRow)),

  /** propertyId comes from context, never a form field — mirrors unitsApi.ts's create(propertyId, ...). */
  create: (propertyId: string, values: RecurringRuleFormValues): Promise<RecurringRuleRow> =>
    apiClient
      .post<RecurringRuleWire>(`/api/v1/properties/${propertyId}/maintenance-rules`, toCreateRecurringRuleRequest(values))
      .then(toRecurringRuleRow),

  update: (id: string, values: RecurringRuleFormValues): Promise<RecurringRuleRow> =>
    apiClient.put<RecurringRuleWire>(`/api/v1/maintenance-rules/${id}`, toCreateRecurringRuleRequest(values)).then(toRecurringRuleRow),

  setActive: (id: string, active: boolean): Promise<RecurringRuleRow> =>
    apiClient.put<RecurringRuleWire>(`/api/v1/maintenance-rules/${id}`, { active }).then(toRecurringRuleRow),

  delete: (id: string): Promise<void> => apiClient.delete<void>(`/api/v1/maintenance-rules/${id}`),

  /** The manager-triggered stand-in for a background scheduler — see the backend's RecurringRule doc comment. Returns the newly generated work order's id. */
  generateNow: (id: string): Promise<{ workOrderId: string }> =>
    apiClient.post<{ id: string }>(`/api/v1/maintenance-rules/${id}/generate`, {}).then((r) => ({ workOrderId: r.id })),
};
