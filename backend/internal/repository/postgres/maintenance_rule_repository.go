package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type MaintenanceRuleRepository struct {
	pool *pgxpool.Pool
}

func NewMaintenanceRuleRepository(pool *pgxpool.Pool) *MaintenanceRuleRepository {
	return &MaintenanceRuleRepository{pool: pool}
}

var _ domain.RecurringRuleRepository = (*MaintenanceRuleRepository)(nil)

const recurringRuleColumns = `
	id, property_id, unit_id, title, description, category, frequency_interval, frequency_unit,
	next_due_date, active, created_at, updated_at`

func (r *MaintenanceRuleRepository) Create(ctx context.Context, rule *domain.RecurringRule) error {
	const q = `
		INSERT INTO maintenance_recurring_rules (` + recurringRuleColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, q,
		rule.ID, rule.PropertyID, rule.UnitID, rule.Title, rule.Description, rule.Category,
		rule.FrequencyInterval, rule.FrequencyUnit, rule.NextDueDate, rule.Active, rule.CreatedAt, rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert recurring rule %s: %w", rule.ID, err)
	}
	return nil
}

func (r *MaintenanceRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RecurringRule, error) {
	const q = `SELECT ` + recurringRuleColumns + ` FROM maintenance_recurring_rules WHERE id = $1`

	rule, err := scanRecurringRule(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get recurring rule %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get recurring rule %s: %w", id, err)
	}
	return rule, nil
}

func (r *MaintenanceRuleRepository) ListForOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.RecurringRuleWithProperty, error) {
	const q = `
		SELECT
			rr.id, rr.property_id, rr.unit_id, rr.title, rr.description, rr.category, rr.frequency_interval,
			rr.frequency_unit, rr.next_due_date, rr.active, rr.created_at, rr.updated_at,
			p.name AS property_name, COALESCE(u.unit_name, '') AS unit_name
		FROM maintenance_recurring_rules rr
		JOIN properties p ON p.id = rr.property_id
		LEFT JOIN units u ON u.id = rr.unit_id
		WHERE p.owner_id = $1
		ORDER BY rr.next_due_date ASC, rr.id ASC`

	rows, err := r.pool.Query(ctx, q, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list recurring rules for owner %s: %w", ownerID, err)
	}
	defer rows.Close()

	rules := make([]*domain.RecurringRuleWithProperty, 0)
	for rows.Next() {
		var rule domain.RecurringRuleWithProperty
		var unitID uuid.NullUUID
		if err := rows.Scan(
			&rule.ID, &rule.PropertyID, &unitID, &rule.Title, &rule.Description, &rule.Category,
			&rule.FrequencyInterval, &rule.FrequencyUnit, &rule.NextDueDate, &rule.Active, &rule.CreatedAt, &rule.UpdatedAt,
			&rule.PropertyName, &rule.UnitName,
		); err != nil {
			return nil, fmt.Errorf("scan recurring-rule-with-property row: %w", err)
		}
		if unitID.Valid {
			v := unitID.UUID
			rule.UnitID = &v
		}
		rules = append(rules, &rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recurring-rule-with-property rows: %w", err)
	}
	return rules, nil
}

func (r *MaintenanceRuleRepository) Update(ctx context.Context, rule *domain.RecurringRule) error {
	const q = `
		UPDATE maintenance_recurring_rules
		SET unit_id = $2, title = $3, description = $4, category = $5, frequency_interval = $6,
			frequency_unit = $7, next_due_date = $8, active = $9, updated_at = $10
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		rule.ID, rule.UnitID, rule.Title, rule.Description, rule.Category,
		rule.FrequencyInterval, rule.FrequencyUnit, rule.NextDueDate, rule.Active, rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update recurring rule %s: %w", rule.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update recurring rule %s: %w", rule.ID, domain.ErrNotFound)
	}
	return nil
}

func (r *MaintenanceRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM maintenance_recurring_rules WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete recurring rule %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete recurring rule %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

func scanRecurringRule(row rowScanner) (*domain.RecurringRule, error) {
	var rule domain.RecurringRule
	var unitID uuid.NullUUID

	err := row.Scan(
		&rule.ID, &rule.PropertyID, &unitID, &rule.Title, &rule.Description, &rule.Category,
		&rule.FrequencyInterval, &rule.FrequencyUnit, &rule.NextDueDate, &rule.Active, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if unitID.Valid {
		v := unitID.UUID
		rule.UnitID = &v
	}
	return &rule, nil
}
