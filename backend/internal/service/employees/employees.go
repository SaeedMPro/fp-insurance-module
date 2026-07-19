// Package employees manages employee master data, dependents, and the
// employee-facing remaining-caps view. Access policy: staff (admin/reviewer)
// see everyone; an employee-role user sees only their own linked record.
package employees

import (
	"context"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

type Repo interface {
	GetEmployee(ctx context.Context, id uuid.UUID) (domain.Employee, error)
	ListEmployees(ctx context.Context, f domain.EmployeeFilter) ([]domain.Employee, int64, error)
	CreateEmployee(ctx context.Context, e *domain.Employee) error
	SaveEmployee(ctx context.Context, e *domain.Employee) error
	ListDependents(ctx context.Context, employeeID uuid.UUID) ([]domain.Dependent, error)
	CreateDependent(ctx context.Context, d *domain.Dependent) error
	GetUser(ctx context.Context, id uuid.UUID) (domain.User, error)
}

// CapsProvider computes per-service-type remaining caps (the coverage service).
type CapsProvider interface {
	RemainingCaps(ctx context.Context, employeeID, planID uuid.UUID, onDate time.Time) ([]domain.RemainingCap, error)
}

type Service struct {
	repo  Repo
	caps  CapsProvider
	clock domain.Clock
}

func NewService(repo Repo, caps CapsProvider, clock domain.Clock) *Service {
	if clock == nil {
		clock = domain.SystemClock{}
	}
	return &Service{repo: repo, caps: caps, clock: clock}
}

// authorizeAccess: staff pass; an employee passes only for their own record.
func (s *Service) authorizeAccess(ctx context.Context, actor domain.Actor, employeeID uuid.UUID) error {
	if actor.IsStaff() {
		return nil
	}
	if actor.Role == domain.RoleEmployee {
		user, err := s.repo.GetUser(ctx, actor.UserID)
		if err == nil && user.EmployeeID != nil && *user.EmployeeID == employeeID {
			return nil
		}
	}
	return domain.Forbiddenf("not permitted to view this employee")
}

func (s *Service) List(ctx context.Context, f domain.EmployeeFilter) ([]domain.Employee, int64, error) {
	return s.repo.ListEmployees(ctx, f)
}

type CreateInput struct {
	PersonnelNo string
	FullName    string
	NationalID  string
	HireDate    time.Time
	Department  string
	PlanID      *uuid.UUID
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Employee, error) {
	e := domain.Employee{
		PersonnelNo:      in.PersonnelNo,
		FullName:         in.FullName,
		NationalID:       in.NationalID,
		EmploymentStatus: domain.EmploymentActive,
		HireDate:         in.HireDate,
		Department:       in.Department,
		PlanID:           in.PlanID,
	}
	if err := s.repo.CreateEmployee(ctx, &e); err != nil {
		return domain.Employee{}, err
	}
	return e, nil
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.Employee, error) {
	if err := s.authorizeAccess(ctx, actor, id); err != nil {
		return domain.Employee{}, err
	}
	return s.repo.GetEmployee(ctx, id)
}

type UpdateInput struct {
	EmploymentStatus *domain.EmploymentStatus
	PlanID           *uuid.UUID
	Department       *string
	FullName         *string
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (domain.Employee, error) {
	e, err := s.repo.GetEmployee(ctx, id)
	if err != nil {
		return domain.Employee{}, err
	}
	if in.EmploymentStatus != nil {
		e.EmploymentStatus = *in.EmploymentStatus
	}
	if in.PlanID != nil {
		e.PlanID = in.PlanID
	}
	if in.Department != nil {
		e.Department = *in.Department
	}
	if in.FullName != nil {
		e.FullName = *in.FullName
	}
	if err := s.repo.SaveEmployee(ctx, &e); err != nil {
		return domain.Employee{}, err
	}
	return e, nil
}

func (s *Service) ListDependents(ctx context.Context, actor domain.Actor, employeeID uuid.UUID) ([]domain.Dependent, error) {
	if err := s.authorizeAccess(ctx, actor, employeeID); err != nil {
		return nil, err
	}
	return s.repo.ListDependents(ctx, employeeID)
}

type CreateDependentInput struct {
	FullName   string
	Relation   domain.Relation
	NationalID string
	BirthDate  *time.Time
}

func (s *Service) CreateDependent(ctx context.Context, employeeID uuid.UUID, in CreateDependentInput) (domain.Dependent, error) {
	d := domain.Dependent{
		EmployeeID: employeeID,
		FullName:   in.FullName,
		Relation:   in.Relation,
		NationalID: in.NationalID,
		BirthDate:  in.BirthDate,
	}
	if err := s.repo.CreateDependent(ctx, &d); err != nil {
		return domain.Dependent{}, err
	}
	return d, nil
}

// RemainingCaps returns the per-service-type cap usage for an employee's plan;
// an employee with no plan gets an empty list.
func (s *Service) RemainingCaps(ctx context.Context, actor domain.Actor, employeeID uuid.UUID) ([]domain.RemainingCap, error) {
	if err := s.authorizeAccess(ctx, actor, employeeID); err != nil {
		return nil, err
	}
	e, err := s.repo.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if e.PlanID == nil {
		return []domain.RemainingCap{}, nil
	}
	return s.caps.RemainingCaps(ctx, e.ID, *e.PlanID, s.clock.Now())
}
