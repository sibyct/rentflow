package domain_test

import (
	"testing"
	"time"

	"propertymanagement/internal/domain"
)

func TestVendor_InsuranceStatus(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		vendor domain.Vendor
		want   domain.InsuranceStatus
	}{
		{
			name:   "no expiry on file is unknown",
			vendor: domain.Vendor{},
			want:   domain.InsuranceStatusUnknown,
		},
		{
			name: "far from expiring is valid",
			vendor: func() domain.Vendor {
				expiry := now.AddDate(0, 6, 0)
				return domain.Vendor{InsuranceExpiry: &expiry}
			}(),
			want: domain.InsuranceStatusValid,
		},
		{
			name: "expires within the expiring-soon window",
			vendor: func() domain.Vendor {
				expiry := now.AddDate(0, 0, 10)
				return domain.Vendor{InsuranceExpiry: &expiry}
			}(),
			want: domain.InsuranceStatusExpiringSoon,
		},
		{
			name: "expires exactly at the expiring-soon boundary",
			vendor: func() domain.Vendor {
				expiry := now.AddDate(0, 0, domain.InsuranceExpiringSoonDays)
				return domain.Vendor{InsuranceExpiry: &expiry}
			}(),
			want: domain.InsuranceStatusExpiringSoon,
		},
		{
			name: "already past expiry is expired",
			vendor: func() domain.Vendor {
				expiry := now.AddDate(0, 0, -1)
				return domain.Vendor{InsuranceExpiry: &expiry}
			}(),
			want: domain.InsuranceStatusExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.vendor.InsuranceStatus(now)
			if got != tt.want {
				t.Errorf("InsuranceStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
