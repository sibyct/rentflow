package domain_test

import (
	"testing"
	"time"

	"propertymanagement/internal/domain"
)

func TestWorkOrder_IsOverdue(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	tests := []struct {
		name string
		w    domain.WorkOrder
		want bool
	}{
		{"no due date is never overdue", domain.WorkOrder{Status: domain.WorkOrderStatusNew}, false},
		{"due date in the past and still open is overdue", domain.WorkOrder{Status: domain.WorkOrderStatusNew, DueDate: &yesterday}, true},
		{"due date in the future is not overdue", domain.WorkOrder{Status: domain.WorkOrderStatusNew, DueDate: &tomorrow}, false},
		{"completed past its due date is not overdue", domain.WorkOrder{Status: domain.WorkOrderStatusCompleted, DueDate: &yesterday}, false},
		{"cancelled past its due date is not overdue", domain.WorkOrder{Status: domain.WorkOrderStatusCancelled, DueDate: &yesterday}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.w.IsOverdue(now); got != tt.want {
				t.Errorf("IsOverdue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecurringRule_NextOccurrence(t *testing.T) {
	base := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		rule domain.RecurringRule
		want time.Time
	}{
		{
			name: "days",
			rule: domain.RecurringRule{NextDueDate: base, FrequencyInterval: 10, FrequencyUnit: domain.RecurringRuleFrequencyDays},
			want: base.AddDate(0, 0, 10),
		},
		{
			name: "weeks",
			rule: domain.RecurringRule{NextDueDate: base, FrequencyInterval: 2, FrequencyUnit: domain.RecurringRuleFrequencyWeeks},
			want: base.AddDate(0, 0, 14),
		},
		{
			name: "months",
			rule: domain.RecurringRule{NextDueDate: base, FrequencyInterval: 3, FrequencyUnit: domain.RecurringRuleFrequencyMonths},
			want: base.AddDate(0, 3, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rule.NextOccurrence(); !got.Equal(tt.want) {
				t.Errorf("NextOccurrence() = %v, want %v", got, tt.want)
			}
		})
	}
}
