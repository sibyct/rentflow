package domain_test

import (
	"testing"
	"time"

	"propertymanagement/internal/domain"
)

func TestLease_DisplayStatus(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		lease domain.Lease
		want  domain.LeaseDisplayStatus
	}{
		{
			name:  "draft stays draft regardless of dates",
			lease: domain.Lease{Status: domain.LeaseStatusDraft, StartDate: now.AddDate(0, -1, 0)},
			want:  domain.LeaseDisplayDraft,
		},
		{
			name:  "terminated stays terminated regardless of dates",
			lease: domain.Lease{Status: domain.LeaseStatusTerminated, StartDate: now.AddDate(0, -1, 0)},
			want:  domain.LeaseDisplayTerminated,
		},
		{
			name:  "starts in the future is upcoming",
			lease: domain.Lease{Status: domain.LeaseStatusActive, StartDate: now.AddDate(0, 0, 5)},
			want:  domain.LeaseDisplayUpcoming,
		},
		{
			name:  "month-to-month with no end date, already started, is active",
			lease: domain.Lease{Status: domain.LeaseStatusActive, StartDate: now.AddDate(0, -6, 0)},
			want:  domain.LeaseDisplayActive,
		},
		{
			name: "fixed-term far from ending is active",
			lease: func() domain.Lease {
				end := now.AddDate(0, 6, 0)
				return domain.Lease{Status: domain.LeaseStatusActive, StartDate: now.AddDate(0, -6, 0), EndDate: &end}
			}(),
			want: domain.LeaseDisplayActive,
		},
		{
			name: "ends within the expiring-soon window",
			lease: func() domain.Lease {
				end := now.AddDate(0, 0, 10)
				return domain.Lease{Status: domain.LeaseStatusActive, StartDate: now.AddDate(0, -6, 0), EndDate: &end}
			}(),
			want: domain.LeaseDisplayExpiringSoon,
		},
		{
			name: "ends exactly at the expiring-soon boundary",
			lease: func() domain.Lease {
				end := now.AddDate(0, 0, domain.LeaseExpiringSoonDays)
				return domain.Lease{Status: domain.LeaseStatusActive, StartDate: now.AddDate(0, -6, 0), EndDate: &end}
			}(),
			want: domain.LeaseDisplayExpiringSoon,
		},
		{
			name: "already past end date is expired",
			lease: func() domain.Lease {
				end := now.AddDate(0, 0, -1)
				return domain.Lease{Status: domain.LeaseStatusActive, StartDate: now.AddDate(0, -6, 0), EndDate: &end}
			}(),
			want: domain.LeaseDisplayExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lease.DisplayStatus(now)
			if got != tt.want {
				t.Errorf("DisplayStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
