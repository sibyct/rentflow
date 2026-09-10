// Mirrors backend/internal/transport/http/dto/property_dto.go

export type PropertyStatus = 'active' | 'inactive' | 'maintenance';

export interface Property {
  id: string;
  address: string;
  unitCount: number;
  status: PropertyStatus;
  ownerId: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreatePropertyInput {
  address: string;
  unitCount: number;
  status?: PropertyStatus;
}

export interface UpdatePropertyInput {
  address?: string;
  unitCount?: number;
  status?: PropertyStatus;
}

export interface PropertyListMeta {
  total: number;
  limit: number;
  offset: number;
}

export interface PropertyListResult {
  properties: Property[];
  meta: PropertyListMeta;
}
