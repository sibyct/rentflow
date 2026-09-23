import { z } from 'zod';
import { WORK_ORDER_CATEGORIES } from '@/features/maintenance/types';
import { VENDOR_PAYMENT_TERMS, VENDOR_RATE_TYPES } from '../types';

// Numeric/date fields are strings in the form (same as leases/work
// orders) and parsed at the API boundary — see vendorsApi.ts's
// toVendorRequest.
export const vendorSchema = z.object({
  companyName: z.string().trim().min(1, 'Company name is required.'),
  categories: z.array(z.enum(WORK_ORDER_CATEGORIES)).min(1, 'Select at least one category.'),
  contactPerson: z.string().trim().optional(),
  phone: z.string().trim().optional(),
  email: z.string().trim().optional(),
  address: z.string().trim().optional(),
  servesAllProperties: z.boolean(),
  propertiesServed: z.array(z.string()),
  insuranceExpiry: z.string().trim().optional(),
  licenseNumber: z.string().trim().optional(),
  licenseExpiry: z.string().trim().optional(),
  coiAttachmentId: z.string().trim().optional(),
  taxDocAttachmentId: z.string().trim().optional(),
  rateType: z.enum(VENDOR_RATE_TYPES).or(z.literal('')).optional(),
  rateAmount: z.string().trim().optional(),
  paymentTerms: z.enum(VENDOR_PAYMENT_TERMS).or(z.literal('')).optional(),
  internalNotes: z.string().trim().optional(),
  active: z.boolean(),
});

export type VendorFormValues = z.infer<typeof vendorSchema>;

export function vendorDefaultValues(): VendorFormValues {
  return {
    companyName: '',
    categories: [],
    contactPerson: '',
    phone: '',
    email: '',
    address: '',
    servesAllProperties: true,
    propertiesServed: [],
    insuranceExpiry: '',
    licenseNumber: '',
    licenseExpiry: '',
    coiAttachmentId: '',
    taxDocAttachmentId: '',
    rateType: '',
    rateAmount: '',
    paymentTerms: '',
    internalNotes: '',
    active: true,
  };
}
