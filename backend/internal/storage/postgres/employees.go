package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"insurance-module/internal/domain"
)

func (s *Store) GetEmployee(ctx context.Context, id uuid.UUID) (domain.Employee, error) {
	var row employeeRow
	if err := s.ctx(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Employee{}, notFound(err, "employee")
	}
	return row.toDomain(), nil
}

// GetEmployeeByPersonnelNo returns (employee, found, err) — the integration
// sync uses "not found" as the create branch, not as an error.
func (s *Store) GetEmployeeByPersonnelNo(ctx context.Context, personnelNo string) (domain.Employee, bool, error) {
	var row employeeRow
	err := s.ctx(ctx).Where("personnel_no = ?", personnelNo).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Employee{}, false, nil
	}
	if err != nil {
		return domain.Employee{}, false, err
	}
	return row.toDomain(), true, nil
}

func (s *Store) ListEmployees(ctx context.Context, f domain.EmployeeFilter) ([]domain.Employee, int64, error) {
	q := s.ctx(ctx).Model(&employeeRow{})
	if f.Query != "" {
		like := "%" + f.Query + "%"
		q = q.Where("full_name ILIKE ? OR personnel_no ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []employeeRow
	if err := q.Order("full_name").Offset(f.Page.Offset()).Limit(f.Page.Size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Employee, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, total, nil
}

func (s *Store) CreateEmployee(ctx context.Context, e *domain.Employee) error {
	row := employeeFromDomain(*e)
	if err := s.ctx(ctx).Create(&row).Error; err != nil {
		return err
	}
	*e = row.toDomain()
	return nil
}

func (s *Store) SaveEmployee(ctx context.Context, e *domain.Employee) error {
	row := employeeFromDomain(*e)
	if err := s.ctx(ctx).Save(&row).Error; err != nil {
		return err
	}
	*e = row.toDomain()
	return nil
}

func (s *Store) ListDependents(ctx context.Context, employeeID uuid.UUID) ([]domain.Dependent, error) {
	var rows []dependentRow
	if err := s.ctx(ctx).Where("employee_id = ?", employeeID).Order("full_name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Dependent, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (s *Store) GetDependent(ctx context.Context, id uuid.UUID) (domain.Dependent, error) {
	var row dependentRow
	if err := s.ctx(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Dependent{}, notFound(err, "dependent")
	}
	return row.toDomain(), nil
}

func (s *Store) CreateDependent(ctx context.Context, d *domain.Dependent) error {
	row := dependentFromDomain(*d)
	if err := s.ctx(ctx).Create(&row).Error; err != nil {
		return err
	}
	*d = row.toDomain()
	return nil
}
