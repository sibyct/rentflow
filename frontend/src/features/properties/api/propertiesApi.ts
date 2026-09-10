import { apiClient } from '@/api/client';
import type {
  CreatePropertyInput,
  Property,
  PropertyListMeta,
  PropertyListResult,
  PropertyStatus,
  UpdatePropertyInput,
} from '../types';

// Wire shape exactly as the Go DTO serializes it (snake_case). Stays
// private to this file; the rest of the app uses the camelCase Property
// type from ../types.
interface PropertyWire {
  id: string;
  address: string;
  unit_count: number;
  status: PropertyStatus;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

function toProperty(wire: PropertyWire): Property {
  return {
    id: wire.id,
    address: wire.address,
    unitCount: wire.unit_count,
    status: wire.status,
    ownerId: wire.owner_id,
    createdAt: wire.created_at,
    updatedAt: wire.updated_at,
  };
}

export interface ListPropertiesParams {
  limit?: number;
  offset?: number;
}

export const propertiesApi = {
  list: async (params: ListPropertiesParams = {}): Promise<PropertyListResult> => {
    const query = new URLSearchParams();
    if (params.limit !== undefined) query.set('limit', String(params.limit));
    if (params.offset !== undefined) query.set('offset', String(params.offset));
    const qs = query.toString();

    const { data, meta } = await apiClient.getWithMeta<PropertyWire[], PropertyListMeta>(
      `/api/v1/properties${qs ? `?${qs}` : ''}`,
    );
    return { properties: data.map(toProperty), meta };
  },

  get: (id: string): Promise<Property> =>
    apiClient.get<PropertyWire>(`/api/v1/properties/${id}`).then(toProperty),

  create: (input: CreatePropertyInput): Promise<Property> =>
    apiClient
      .post<PropertyWire>('/api/v1/properties', {
        address: input.address,
        unit_count: input.unitCount,
        status: input.status,
      })
      .then(toProperty),

  update: (id: string, input: UpdatePropertyInput): Promise<Property> =>
    apiClient
      .put<PropertyWire>(`/api/v1/properties/${id}`, {
        address: input.address,
        unit_count: input.unitCount,
        status: input.status,
      })
      .then(toProperty),

  delete: (id: string): Promise<void> => apiClient.delete<void>(`/api/v1/properties/${id}`),
};
