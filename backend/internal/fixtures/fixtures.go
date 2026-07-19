// Package fixtures creates the demo data used for local development, grading
// demos, and the E2E suite: one user per role, sample employees/dependents,
// and claims walked through different stages of the workflow via the real
// services (so audit entries and pricing are genuine). Seeding is idempotent —
// every record is looked up by its natural key before being created.
package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/claims"
	"insurance-module/internal/service/employees"
	"insurance-module/internal/service/users"
	"insurance-module/internal/storage/postgres"
	transporthttp "insurance-module/internal/transport/http"
)

// DemoAccounts is printed by cmd/seed so graders can log in.
const DemoAccounts = `seed complete. Demo accounts (username / password):
  admin / Admin123!
  reviewer / Reviewer123!
  auditor / Auditor123!
  sara.ahmadi / Employee123!
  reza.karimi / Employee123!`

// Seed loads the demo fixtures through the service layer.
func Seed(ctx context.Context, store *postgres.Store, svcs transporthttp.Services) error {
	standardPlan, err := store.GetPlanByName(ctx, "استاندارد")
	if err != nil {
		return fmt.Errorf("plan lookup (run migrations first): %w", err)
	}
	premiumPlan, err := store.GetPlanByName(ctx, "ویژه")
	if err != nil {
		return fmt.Errorf("plan lookup: %w", err)
	}

	if _, err := ensureUser(ctx, store, svcs.Users, "admin", "Admin123!", "مدیر سامانه", domain.RoleAdmin, nil); err != nil {
		return err
	}
	reviewer, err := ensureUser(ctx, store, svcs.Users, "reviewer", "Reviewer123!", "کارشناس بررسی خسارت", domain.RoleReviewer, nil)
	if err != nil {
		return err
	}
	if _, err := ensureUser(ctx, store, svcs.Users, "auditor", "Auditor123!", "ممیز انطباق", domain.RoleAuditor, nil); err != nil {
		return err
	}

	emp1, err := ensureEmployee(ctx, store, svcs.Employees, "P-1001", "سارا احمدی", "0011223344", time.Now().AddDate(-3, 0, 0), "مهندسی", standardPlan.ID)
	if err != nil {
		return err
	}
	emp2, err := ensureEmployee(ctx, store, svcs.Employees, "P-1002", "رضا کریمی", "0022334455", time.Now().AddDate(-1, -6, 0), "مالی", premiumPlan.ID)
	if err != nil {
		return err
	}

	empUser1, err := ensureUser(ctx, store, svcs.Users, "sara.ahmadi", "Employee123!", emp1.FullName, domain.RoleEmployee, &emp1.ID)
	if err != nil {
		return err
	}
	empUser2, err := ensureUser(ctx, store, svcs.Users, "reza.karimi", "Employee123!", emp2.FullName, domain.RoleEmployee, &emp2.ID)
	if err != nil {
		return err
	}

	dep1, err := ensureDependent(ctx, store, svcs.Employees, emp1.ID, "نیلوفر احمدی", domain.RelationChild, time.Now().AddDate(-8, 0, 0))
	if err != nil {
		return err
	}

	return seedDemoClaims(ctx, store, svcs.Claims, demoActors{
		reviewer: actorOf(reviewer),
		emp1:     actorOf(empUser1),
		emp2:     actorOf(empUser2),
	}, emp1, emp2, dep1)
}

func actorOf(u domain.User) domain.Actor {
	return domain.Actor{UserID: u.ID, Username: u.Username, Role: u.Role}
}

func ensureUser(ctx context.Context, store *postgres.Store, svc *users.Service, username, password, fullName string, role domain.Role, employeeID *uuid.UUID) (domain.User, error) {
	if existing, err := store.GetUserByUsername(ctx, username); err == nil {
		return existing, nil
	}
	return svc.Create(ctx, users.CreateInput{
		Username: username, Password: password, FullName: fullName,
		Role: role, EmployeeID: employeeID,
	})
}

func ensureEmployee(ctx context.Context, store *postgres.Store, svc *employees.Service, personnelNo, fullName, nationalID string, hireDate time.Time, department string, planID uuid.UUID) (domain.Employee, error) {
	if existing, found, err := store.GetEmployeeByPersonnelNo(ctx, personnelNo); err != nil {
		return domain.Employee{}, err
	} else if found {
		return existing, nil
	}
	return svc.Create(ctx, employees.CreateInput{
		PersonnelNo: personnelNo, FullName: fullName, NationalID: nationalID,
		HireDate: hireDate, Department: department, PlanID: &planID,
	})
}

func ensureDependent(ctx context.Context, store *postgres.Store, svc *employees.Service, employeeID uuid.UUID, fullName string, relation domain.Relation, birthDate time.Time) (domain.Dependent, error) {
	existing, err := store.ListDependents(ctx, employeeID)
	if err != nil {
		return domain.Dependent{}, err
	}
	for _, d := range existing {
		if d.FullName == fullName {
			return d, nil
		}
	}
	return svc.CreateDependent(ctx, employeeID, employees.CreateDependentInput{
		FullName: fullName, Relation: relation, BirthDate: &birthDate,
	})
}

type demoActors struct {
	reviewer domain.Actor
	emp1     domain.Actor
	emp2     domain.Actor
}

// seedDemoClaims walks four claims to different workflow stages (closed,
// under review, rejected, freshly submitted) so every screen has data.
// Skipped entirely if any claims already exist.
func seedDemoClaims(ctx context.Context, store *postgres.Store, svc *claims.Service, actors demoActors, emp1, emp2 domain.Employee, dep1 domain.Dependent) error {
	_, total, err := store.ListClaims(ctx, domain.ClaimFilter{Page: domain.NewPage(1, 1)})
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}

	outpatient, err := store.GetServiceTypeByCode(ctx, "outpatient_visit")
	if err != nil {
		return err
	}
	pharmacy, err := store.GetServiceTypeByCode(ctx, "pharmacy")
	if err != nil {
		return err
	}
	dental, err := store.GetServiceTypeByCode(ctx, "dental")
	if err != nil {
		return err
	}

	// Claim 1: the full happy path through to closed.
	c1, err := svc.Create(ctx, actors.emp1, claims.CreateInput{
		BeneficiaryType: domain.BeneficiarySelf, ServiceTypeID: outpatient.ID,
		RequestedAmount: 350000, ReceiptDate: time.Now().AddDate(0, 0, -20),
		Description: "ویزیت سرپایی - معاینه سالانه",
	})
	if err != nil {
		return err
	}
	for _, step := range []func() error{
		func() error { _, err := svc.Submit(ctx, actors.emp1, c1.ID); return err },
		func() error { _, err := svc.StartReview(ctx, actors.reviewer, c1.ID); return err },
		func() error { _, err := svc.Approve(ctx, actors.reviewer, c1.ID); return err },
		func() error { _, err := svc.MarkPaid(ctx, actors.reviewer, c1.ID); return err },
		func() error { _, err := svc.Close(ctx, actors.reviewer, c1.ID); return err },
	} {
		if err := step(); err != nil {
			return fmt.Errorf("seed claim 1: %w", err)
		}
	}

	// Claim 2 (dependent beneficiary): sitting under review.
	c2, err := svc.Create(ctx, actors.emp1, claims.CreateInput{
		BeneficiaryType: domain.BeneficiaryDependent, DependentID: &dep1.ID,
		ServiceTypeID: dental.ID, RequestedAmount: 1200000,
		ReceiptDate: time.Now().AddDate(0, 0, -5), Description: "پر کردن دندان عضو تحت تکفل",
	})
	if err != nil {
		return err
	}
	if _, err := svc.Submit(ctx, actors.emp1, c2.ID); err != nil {
		return err
	}
	if _, err := svc.StartReview(ctx, actors.reviewer, c2.ID); err != nil {
		return err
	}

	// Claim 3: rejected (missing prescription).
	c3, err := svc.Create(ctx, actors.emp2, claims.CreateInput{
		BeneficiaryType: domain.BeneficiarySelf, ServiceTypeID: pharmacy.ID,
		RequestedAmount: 200000, ReceiptDate: time.Now().AddDate(0, 0, -15),
		Description: "خرید دارو - بدون نسخه",
	})
	if err != nil {
		return err
	}
	if _, err := svc.Submit(ctx, actors.emp2, c3.ID); err != nil {
		return err
	}
	if _, err := svc.StartReview(ctx, actors.reviewer, c3.ID); err != nil {
		return err
	}
	if _, err := svc.Reject(ctx, actors.reviewer, c3.ID, "نسخه پزشک پیوست نشده است"); err != nil {
		return err
	}

	// Claim 4: freshly submitted — for the reviewer queue demo.
	c4, err := svc.Create(ctx, actors.emp2, claims.CreateInput{
		BeneficiaryType: domain.BeneficiarySelf, ServiceTypeID: outpatient.ID,
		RequestedAmount: 500000, ReceiptDate: time.Now(),
		Description: "ویزیت سرپایی - پیگیری",
	})
	if err != nil {
		return err
	}
	if _, err := svc.Submit(ctx, actors.emp2, c4.ID); err != nil {
		return err
	}
	return nil
}
