import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  accountingApi,
  type ChargeListParams,
  type ExpenseListParams,
  type RentRollParams,
} from '../api/accountingApi';
import type { ChargeFormValues } from '../schemas/chargeSchema';
import type { ExpenseFormValues } from '../schemas/expenseSchema';
import type { PaymentFormValues } from '../schemas/paymentSchema';
import type { AccountingSettings, DashboardPeriod } from '../types';

export const accountingQueryKeys = {
  all: ['accounting'] as const,
  dashboard: (period: DashboardPeriod) => [...accountingQueryKeys.all, 'dashboard', period] as const,
  settings: () => [...accountingQueryKeys.all, 'settings'] as const,
  rentRoll: (params: RentRollParams) => [...accountingQueryKeys.all, 'rent-roll', params] as const,
  expenses: (params: ExpenseListParams) => [...accountingQueryKeys.all, 'expenses', params] as const,
  expense: (id: string) => [...accountingQueryKeys.all, 'expense', id] as const,
  charges: (params: ChargeListParams) => [...accountingQueryKeys.all, 'charges', params] as const,
  payments: (transactionId: string) => [...accountingQueryKeys.all, 'payments', transactionId] as const,
  audit: (transactionId: string) => [...accountingQueryKeys.all, 'audit', transactionId] as const,
};

export function useAccountingDashboard(period: DashboardPeriod) {
  return useQuery({
    queryKey: accountingQueryKeys.dashboard(period),
    queryFn: () => accountingApi.dashboard(period),
    placeholderData: keepPreviousData,
  });
}

export function useAccountingSettings() {
  return useQuery({ queryKey: accountingQueryKeys.settings(), queryFn: () => accountingApi.getSettings() });
}

export function useRentRoll(params: RentRollParams) {
  return useQuery({
    queryKey: accountingQueryKeys.rentRoll(params),
    queryFn: () => accountingApi.listRentRoll(params),
    placeholderData: keepPreviousData,
  });
}

export function useExpenses(params: ExpenseListParams) {
  return useQuery({
    queryKey: accountingQueryKeys.expenses(params),
    queryFn: () => accountingApi.listExpenses(params),
    placeholderData: keepPreviousData,
  });
}

export function useExpense(id: string | undefined) {
  return useQuery({
    queryKey: accountingQueryKeys.expense(id ?? ''),
    queryFn: () => accountingApi.getExpense(id!),
    enabled: Boolean(id),
  });
}

export function useCharges(params: ChargeListParams) {
  return useQuery({
    queryKey: accountingQueryKeys.charges(params),
    queryFn: () => accountingApi.listCharges(params),
    placeholderData: keepPreviousData,
  });
}

export function usePayments(transactionId: string | undefined) {
  return useQuery({
    queryKey: accountingQueryKeys.payments(transactionId ?? ''),
    queryFn: () => accountingApi.listPayments(transactionId!),
    enabled: Boolean(transactionId),
  });
}

export function useTransactionAudit(transactionId: string | undefined) {
  return useQuery({
    queryKey: accountingQueryKeys.audit(transactionId ?? ''),
    queryFn: () => accountingApi.listAudit(transactionId!),
    enabled: Boolean(transactionId),
  });
}

/**
 * Every accounting mutation invalidates the whole `accounting` key tree:
 * a payment moves the Rent Roll, the Expenses/Charges lists and the
 * dashboard KPIs at once, and they all hang off this one root (which is
 * also what the main Dashboard's finance cards read from).
 */
function useInvalidateAccounting() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: accountingQueryKeys.all });
}

export function useUpdateAccountingSettings() {
  const invalidate = useInvalidateAccounting();
  return useMutation({
    mutationFn: (s: AccountingSettings) => accountingApi.updateSettings(s),
    onSuccess: invalidate,
  });
}

export function useGenerateRent() {
  const invalidate = useInvalidateAccounting();
  return useMutation({ mutationFn: (period: string) => accountingApi.generateRent(period), onSuccess: invalidate });
}

export function useRecordLeasePayment() {
  const invalidate = useInvalidateAccounting();
  return useMutation({
    mutationFn: ({ leaseId, values }: { leaseId: string; values: PaymentFormValues }) =>
      accountingApi.recordLeasePayment(leaseId, values),
    onSuccess: invalidate,
  });
}

export function useRecordPayment() {
  const invalidate = useInvalidateAccounting();
  return useMutation({
    mutationFn: ({ transactionId, values }: { transactionId: string; values: PaymentFormValues }) =>
      accountingApi.recordPayment(transactionId, values),
    onSuccess: invalidate,
  });
}

export function useCreateExpense() {
  const invalidate = useInvalidateAccounting();
  return useMutation({ mutationFn: (values: ExpenseFormValues) => accountingApi.createExpense(values), onSuccess: invalidate });
}

export function useUpdateExpense(id: string) {
  const invalidate = useInvalidateAccounting();
  return useMutation({
    mutationFn: (values: ExpenseFormValues) => accountingApi.updateExpense(id, values),
    onSuccess: invalidate,
  });
}

export function useCreateCharge() {
  const invalidate = useInvalidateAccounting();
  return useMutation({ mutationFn: (values: ChargeFormValues) => accountingApi.createCharge(values), onSuccess: invalidate });
}

export function useVoidTransaction() {
  const invalidate = useInvalidateAccounting();
  return useMutation({ mutationFn: (id: string) => accountingApi.voidTransaction(id), onSuccess: invalidate });
}

export function useVoidPayment() {
  const invalidate = useInvalidateAccounting();
  return useMutation({ mutationFn: (id: string) => accountingApi.voidPayment(id), onSuccess: invalidate });
}
