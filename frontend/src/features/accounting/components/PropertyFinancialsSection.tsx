import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import ArrowForwardOutlined from '@mui/icons-material/ArrowForwardOutlined';
import Button from '@mui/material/Button';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import type { PropertyRow } from '@/features/properties/mock/propertyRows';
import { ErrorBoundary, ErrorMessage } from '@/shared/components';
import { currentMonthIso, dollarsToCents, formatMoney } from '@/shared/lib/format';
import { useExpenses, useRecordLeasePayment, useRentRoll } from '../hooks/useAccountingQueries';
import type { ExpenseSortKey, RentRollSortKey } from '../api/accountingApi';
import type { LedgerRow, RentRollRow } from '../types';
import { ExpenseDrawer } from './ExpenseDrawer';
import { ExpensesTable, type ExpensesEmptyState } from './ExpensesTable';
import { RecordPaymentDialog } from './RecordPaymentDialog';
import { RentRollTable, type RentRollEmptyState } from './RentRollTable';

interface PropertyFinancialsSectionProps {
  property: PropertyRow;
}

function errorMessage(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Something went wrong. Please try again.';
}

/**
 * Property-scoped zoom level of Accounting — the current month's rent
 * roll and this property's expenses, both fixed to this property, no
 * property picker. Mirrors UnitsSection's role for Units; unlike the
 * other property-scoped sections, this one stacks two cards (Rent Roll,
 * Expenses) since neither is the "whole tab" on its own the way one
 * table is for Units/Leases/Maintenance.
 */
export function PropertyFinancialsSection({ property }: PropertyFinancialsSectionProps) {
  const propertyId = property.id;
  const month = currentMonthIso();

  const [rentSortKey, setRentSortKey] = useState<RentRollSortKey>('unit');
  const [rentSortDir, setRentSortDir] = useState<'asc' | 'desc'>('asc');
  const [payTarget, setPayTarget] = useState<RentRollRow | null>(null);
  const [payError, setPayError] = useState<string | null>(null);

  const [expenseSortKey, setExpenseSortKey] = useState<ExpenseSortKey>('incurredOn');
  const [expenseSortDir, setExpenseSortDir] = useState<'asc' | 'desc'>('desc');
  const [expensePage, setExpensePage] = useState(0);
  const [expenseDrawer, setExpenseDrawer] = useState<{ open: boolean; expenseId?: string }>({ open: false });

  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const rentRoll = useRentRoll({
    propertyId,
    from: month,
    to: month,
    sort: rentSortKey,
    order: rentSortDir,
    limit: 50,
    offset: 0,
  });
  const recordPayment = useRecordLeasePayment();

  const expenses = useExpenses({
    propertyId,
    sort: expenseSortKey,
    order: expenseSortDir,
    limit: 10,
    offset: expensePage * 10,
  });

  const rentRows = rentRoll.data?.rows ?? [];
  const rentTotal = rentRoll.data?.total ?? 0;
  const rentEmptyState: RentRollEmptyState = rentRoll.isLoading || rentTotal > 0 ? { kind: 'none' } : { kind: 'first-time' };

  const expenseRows = expenses.data?.rows ?? [];
  const expenseTotal = expenses.data?.total ?? 0;
  const expenseEmptyState: ExpensesEmptyState = expenses.isLoading || expenseTotal > 0 ? { kind: 'none' } : { kind: 'first-time' };

  function handleRentSort(key: RentRollSortKey) {
    if (rentSortKey === key) setRentSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setRentSortKey(key);
      setRentSortDir('asc');
    }
  }

  function handleExpenseSort(key: ExpenseSortKey) {
    if (expenseSortKey === key) setExpenseSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setExpenseSortKey(key);
      setExpenseSortDir('asc');
    }
    setExpensePage(0);
  }

  return (
    <Stack spacing={2.5}>
      <Paper variant="outlined" sx={{ p: 3 }}>
        <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2, flexWrap: 'wrap', gap: 1 }}>
          <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Rent roll — this month</Typography>
          <Link
            component={RouterLink}
            to={`/accounting/rent-roll?property=${propertyId}`}
            sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, fontSize: 12.5, fontWeight: 600 }}
          >
            View in Rent Roll <ArrowForwardOutlined sx={{ fontSize: 15 }} />
          </Link>
        </Stack>

        {rentRoll.isError ? (
          <ErrorMessage error={rentRoll.error} onRetry={() => void rentRoll.refetch()} />
        ) : (
          <ErrorBoundary>
            <RentRollTable
              loading={rentRoll.isLoading}
              rows={rentRows}
              total={rentTotal}
              emptyState={rentEmptyState}
              sortKey={rentSortKey}
              sortDir={rentSortDir}
              onSort={handleRentSort}
              page={0}
              rowsPerPage={50}
              onPageChange={() => {}}
              onRowsPerPageChange={() => {}}
              onResetView={() => {}}
              onRecordPayment={(r) => {
                setPayError(null);
                setPayTarget(r);
              }}
            />
          </ErrorBoundary>
        )}
      </Paper>

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2, flexWrap: 'wrap', gap: 1 }}>
          <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Expenses</Typography>
          <Stack direction="row" spacing={2.5} sx={{ alignItems: 'center' }}>
            <Link
              component={RouterLink}
              to={`/accounting/expenses?property=${propertyId}`}
              sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, fontSize: 12.5, fontWeight: 600 }}
            >
              View in Expenses <ArrowForwardOutlined sx={{ fontSize: 15 }} />
            </Link>
            <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={() => setExpenseDrawer({ open: true })}>
              Add Expense
            </Button>
          </Stack>
        </Stack>

        {expenses.isError ? (
          <ErrorMessage error={expenses.error} onRetry={() => void expenses.refetch()} />
        ) : (
          <ErrorBoundary>
            <ExpensesTable
              loading={expenses.isLoading}
              rows={expenseRows}
              total={expenseTotal}
              emptyState={expenseEmptyState}
              sortKey={expenseSortKey}
              sortDir={expenseSortDir}
              onSort={handleExpenseSort}
              page={expensePage}
              rowsPerPage={10}
              onPageChange={setExpensePage}
              onRowsPerPageChange={() => {}}
              onAdd={() => setExpenseDrawer({ open: true })}
              onResetView={() => setExpensePage(0)}
              onOpen={(row: LedgerRow) => setExpenseDrawer({ open: true, expenseId: row.id })}
            />
          </ErrorBoundary>
        )}
      </Paper>

      <RecordPaymentDialog
        open={Boolean(payTarget)}
        title="Record payment"
        subtitle={payTarget ? `${payTarget.unitName} — ${payTarget.tenantName}` : undefined}
        outstandingCents={payTarget?.outstandingCents ?? 0}
        submitting={recordPayment.isPending}
        error={payError}
        onClose={() => setPayTarget(null)}
        onSubmit={(values) => {
          if (!payTarget) return;
          recordPayment.mutate(
            { leaseId: payTarget.leaseId, values },
            {
              onSuccess: () => {
                setToast({ message: `Payment of ${formatMoney(dollarsToCents(values.amount) ?? 0)} recorded`, severity: 'success' });
                setPayTarget(null);
              },
              onError: (err) => setPayError(errorMessage(err)),
            },
          );
        }}
      />

      <ExpenseDrawer
        open={expenseDrawer.open}
        onClose={() => setExpenseDrawer({ open: false })}
        properties={[property]}
        expenseId={expenseDrawer.expenseId}
        onSaved={(message) => {
          setExpenseDrawer({ open: false });
          setToast({ message, severity: 'success' });
        }}
      />

      <Snackbar open={Boolean(toast)} autoHideDuration={3200} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {toast ? (
          <Alert onClose={() => setToast(null)} severity={toast.severity} variant="filled" sx={{ borderRadius: 999 }}>
            {toast.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Stack>
  );
}
