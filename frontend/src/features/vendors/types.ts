// Display types for the Vendors feature. Categories reuse
// WorkOrderCategory from features/maintenance/types so a vendor's
// trades line up 1:1 with the work order categories they're matched
// against — see VendorSelector.
import type { WorkOrderCategory } from '@/features/maintenance/types';
export type { WorkOrderCategory } from '@/features/maintenance/types';
export { WORK_ORDER_CATEGORIES, WORK_ORDER_CATEGORY_LABELS } from '@/features/maintenance/types';

export type VendorRateType = 'hourly' | 'flat';

export const VENDOR_RATE_TYPES: VendorRateType[] = ['hourly', 'flat'];

export const VENDOR_RATE_TYPE_LABELS: Record<VendorRateType, string> = {
  hourly: 'Hourly',
  flat: 'Flat',
};

export type VendorPaymentTerms = 'net_15' | 'net_30' | 'net_45';

export const VENDOR_PAYMENT_TERMS: VendorPaymentTerms[] = ['net_15', 'net_30', 'net_45'];

export const VENDOR_PAYMENT_TERMS_LABELS: Record<VendorPaymentTerms, string> = {
  net_15: 'Net 15',
  net_30: 'Net 30',
  net_45: 'Net 45',
};

export type InsuranceStatus = 'valid' | 'expiring_soon' | 'expired' | 'unknown';

export const INSURANCE_STATUS_LABELS: Record<InsuranceStatus, string> = {
  valid: 'Valid',
  expiring_soon: 'Expiring soon',
  expired: 'Expired',
  unknown: 'Unknown',
};

export type VendorSortKey = 'name' | 'rating';

/** Full vendor record — list rows and the profile page both show every field, so unlike Units/Leases/WorkOrders there's no separate row/detail split. */
export interface VendorRow {
  id: string;
  companyName: string;
  categories: WorkOrderCategory[];
  contactPerson: string;
  phone: string;
  email: string;
  address: string;
  servesAllProperties: boolean;
  insuranceExpiry: string;
  insuranceStatus: InsuranceStatus;
  licenseNumber: string;
  licenseExpiry: string;
  coiAttachmentId: string;
  taxDocAttachmentId: string;
  rateType: VendorRateType | '';
  rateAmount: number | null;
  paymentTerms: VendorPaymentTerms | '';
  internalNotes: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
  /** Open (non-terminal) work orders currently assigned to this vendor. */
  openWorkOrders: number;
  /** Average of every rated, completed work order — null until at least one exists. */
  averageRating: number | null;
  /** Meaningful only when !servesAllProperties. */
  propertiesServedCount: number;
}

export type VendorDetail = VendorRow;

export interface VendorSpendSummary {
  thisMonth: number;
  yearToDate: number;
}
