import type { ReactElement } from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { PropertyCard } from './PropertyCard';
import type { Property } from '../types';

const property: Property = {
  id: 'a1b2c3d4-0000-0000-0000-000000000000',
  address: '123 Main St',
  unitCount: 4,
  status: 'active',
  ownerId: 'owner-1',
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
};

function renderWithRouter(ui: ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe('PropertyCard', () => {
  it('renders the address and unit count', () => {
    renderWithRouter(<PropertyCard property={property} />);

    expect(screen.getByText('123 Main St')).toBeInTheDocument();
    expect(screen.getByText('4 units')).toBeInTheDocument();
  });

  it('renders singular "unit" for a single-unit property', () => {
    renderWithRouter(<PropertyCard property={{ ...property, unitCount: 1 }} />);

    expect(screen.getByText('1 unit')).toBeInTheDocument();
  });

  it('links to the property detail page', () => {
    renderWithRouter(<PropertyCard property={property} />);

    expect(screen.getByRole('link')).toHaveAttribute('href', `/properties/${property.id}`);
  });

  it('shows the property status', () => {
    renderWithRouter(<PropertyCard property={{ ...property, status: 'maintenance' }} />);

    expect(screen.getByText('maintenance')).toBeInTheDocument();
  });
});
