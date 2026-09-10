import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { PropertyForm } from './PropertyForm';

describe('PropertyForm', () => {
  it('shows a field-level error on blur, before submitting', async () => {
    const user = userEvent.setup();
    render(<PropertyForm onSubmit={vi.fn()} />);

    await user.click(screen.getByLabelText('Address'));
    await user.tab(); // blur without typing anything

    expect(await screen.findByText(/address must be at least 3 characters/i)).toBeInTheDocument();
  });

  it('clears the field error once the value becomes valid', async () => {
    const user = userEvent.setup();
    render(<PropertyForm onSubmit={vi.fn()} />);

    const addressField = screen.getByLabelText('Address');
    await user.click(addressField);
    await user.tab();
    expect(await screen.findByText(/address must be at least 3 characters/i)).toBeInTheDocument();

    await user.type(addressField, '123 Main St');
    await waitFor(() => {
      expect(screen.queryByText(/address must be at least 3 characters/i)).not.toBeInTheDocument();
    });
  });

  it('calls onSubmit with validated values once the form is valid', async () => {
    const user = userEvent.setup();
    const handleSubmit = vi.fn();
    render(<PropertyForm onSubmit={handleSubmit} />);

    await user.type(screen.getByLabelText('Address'), '123 Main St');
    await user.click(screen.getByRole('button', { name: /create property/i }));

    await waitFor(() => {
      expect(handleSubmit).toHaveBeenCalledWith({ address: '123 Main St', unitCount: 1, status: 'active' });
    });
  });

  it('does not call onSubmit while the form is invalid', async () => {
    const user = userEvent.setup();
    const handleSubmit = vi.fn();
    render(<PropertyForm onSubmit={handleSubmit} />);

    await user.click(screen.getByRole('button', { name: /create property/i }));

    await screen.findByText(/address must be at least 3 characters/i);
    expect(handleSubmit).not.toHaveBeenCalled();
  });
});
