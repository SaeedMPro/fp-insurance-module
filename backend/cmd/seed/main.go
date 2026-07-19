// Command seed populates demo data for local development and grading demos:
// one user per role, a handful of employees/dependents, and a few claims
// walked through different points in the workflow. It is idempotent — safe to
// run more than once — because every row is looked up by its natural key
// (username / personnel_no) before being created.
package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"insurance-module/internal/audit"
	"insurance-module/internal/auth"
	"insurance-module/internal/config"
	"insurance-module/internal/db"
	"insurance-module/internal/models"
	"insurance-module/internal/ruleengine"
	"insurance-module/internal/workflow"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	gdb, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	ctx := context.Background()

	standardPlan := mustPlan(gdb, "استاندارد")
	premiumPlan := mustPlan(gdb, "ویژه")

	upsertUser(gdb, "admin", "Admin123!", "مدیر سامانه", models.RoleAdmin, nil)
	reviewer := upsertUser(gdb, "reviewer", "Reviewer123!", "کارشناس بررسی خسارت", models.RoleReviewer, nil)
	upsertUser(gdb, "auditor", "Auditor123!", "ممیز انطباق", models.RoleAuditor, nil)

	emp1 := upsertEmployee(gdb, "P-1001", "سارا احمدی", "0011223344", time.Now().AddDate(-3, 0, 0), "مهندسی", standardPlan.ID)
	emp2 := upsertEmployee(gdb, "P-1002", "رضا کریمی", "0022334455", time.Now().AddDate(-1, -6, 0), "مالی", premiumPlan.ID)

	upsertUser(gdb, "sara.ahmadi", "Employee123!", emp1.FullName, models.RoleEmployee, &emp1.ID)
	upsertUser(gdb, "reza.karimi", "Employee123!", emp2.FullName, models.RoleEmployee, &emp2.ID)

	dep1 := upsertDependent(gdb, emp1.ID, "نیلوفر احمدی", models.RelationChild, time.Now().AddDate(-8, 0, 0))

	rules := ruleengine.NewEngine(gdb)
	auditSvc := audit.NewService(gdb)
	wf := workflow.NewEngine(gdb, rules, auditSvc)

	seedDemoClaims(ctx, gdb, wf, emp1, emp2, dep1, reviewer)

	log.Println("seed complete. Demo accounts (username / password):")
	log.Println("  admin / Admin123!")
	log.Println("  reviewer / Reviewer123!")
	log.Println("  auditor / Auditor123!")
	log.Println("  sara.ahmadi / Employee123!")
	log.Println("  reza.karimi / Employee123!")
}

func mustPlan(gdb *gorm.DB, name string) models.CoveragePlan {
	var plan models.CoveragePlan
	if err := gdb.Where("name = ?", name).First(&plan).Error; err != nil {
		log.Fatalf("plan %q not found — run migrations first: %v", name, err)
	}
	return plan
}

func upsertUser(gdb *gorm.DB, username, password, fullName string, role models.Role, employeeID *uuid.UUID) models.User {
	var u models.User
	err := gdb.Where("username = ?", username).First(&u).Error
	if err == nil {
		return u
	}
	hash, hErr := auth.HashPassword(password)
	if hErr != nil {
		log.Fatalf("hash password for %s: %v", username, hErr)
	}
	u = models.User{
		Username:     username,
		PasswordHash: hash,
		FullName:     fullName,
		Role:         role,
		EmployeeID:   employeeID,
		IsActive:     true,
	}
	if err := gdb.Create(&u).Error; err != nil {
		log.Fatalf("create user %s: %v", username, err)
	}
	return u
}

func upsertEmployee(gdb *gorm.DB, personnelNo, fullName, nationalID string, hireDate time.Time, department string, planID uuid.UUID) models.Employee {
	var e models.Employee
	err := gdb.Where("personnel_no = ?", personnelNo).First(&e).Error
	if err == nil {
		return e
	}
	e = models.Employee{
		PersonnelNo:      personnelNo,
		FullName:         fullName,
		NationalID:       nationalID,
		EmploymentStatus: models.EmploymentActive,
		HireDate:         hireDate,
		Department:       department,
		PlanID:           &planID,
	}
	if err := gdb.Create(&e).Error; err != nil {
		log.Fatalf("create employee %s: %v", personnelNo, err)
	}
	return e
}

func upsertDependent(gdb *gorm.DB, employeeID uuid.UUID, fullName string, relation models.Relation, birthDate time.Time) models.Dependent {
	var d models.Dependent
	err := gdb.Where("employee_id = ? AND full_name = ?", employeeID, fullName).First(&d).Error
	if err == nil {
		return d
	}
	d = models.Dependent{EmployeeID: employeeID, FullName: fullName, Relation: relation, BirthDate: &birthDate}
	if err := gdb.Create(&d).Error; err != nil {
		log.Fatalf("create dependent %s: %v", fullName, err)
	}
	return d
}

func seedDemoClaims(ctx context.Context, gdb *gorm.DB, wf *workflow.Engine, emp1, emp2 models.Employee, dep1 models.Dependent, reviewer models.User) {
	var count int64
	gdb.Model(&models.Claim{}).Count(&count)
	if count > 0 {
		return
	}

	var outpatient, pharmacy, dental models.ServiceType
	gdb.Where("code = ?", "outpatient_visit").First(&outpatient)
	gdb.Where("code = ?", "pharmacy").First(&pharmacy)
	gdb.Where("code = ?", "dental").First(&dental)

	var empUser1 models.User
	gdb.Where("employee_id = ?", emp1.ID).First(&empUser1)
	var empUser2 models.User
	gdb.Where("employee_id = ?", emp2.ID).First(&empUser2)

	reviewerActor := workflow.Actor{UserID: reviewer.ID, Username: reviewer.Username, Role: reviewer.Role}
	emp1Actor := workflow.Actor{UserID: empUser1.ID, Username: empUser1.Username, Role: empUser1.Role}
	emp2Actor := workflow.Actor{UserID: empUser2.ID, Username: empUser2.Username, Role: empUser2.Role}

	// Claim 1: full happy path through to closed.
	c1 := models.Claim{
		EmployeeID: emp1.ID, BeneficiaryType: models.BeneficiarySelf, ServiceTypeID: outpatient.ID,
		PlanID: *emp1.PlanID, RequestedAmount: 350000, ReceiptDate: time.Now().AddDate(0, 0, -20),
		Description: "ویزیت سرپایی - معاینه سالانه", Status: models.ClaimDraft, CreatedBy: empUser1.ID,
	}
	gdb.Create(&c1)
	mustTransition(wf.Submit(ctx, emp1Actor, c1.ID))
	mustTransition(wf.StartReview(ctx, reviewerActor, c1.ID))
	mustTransition(wf.Approve(ctx, reviewerActor, c1.ID))
	mustTransition(wf.MarkPaid(ctx, reviewerActor, c1.ID))
	mustTransition(wf.Close(ctx, reviewerActor, c1.ID))

	// Claim 2 (dependent beneficiary): sitting under review, awaiting a decision.
	c2 := models.Claim{
		EmployeeID: emp1.ID, BeneficiaryType: models.BeneficiaryDependent, DependentID: &dep1.ID,
		ServiceTypeID: dental.ID, PlanID: *emp1.PlanID, RequestedAmount: 1200000,
		ReceiptDate: time.Now().AddDate(0, 0, -5), Description: "پر کردن دندان عضو تحت تکفل",
		Status: models.ClaimDraft, CreatedBy: empUser1.ID,
	}
	gdb.Create(&c2)
	mustTransition(wf.Submit(ctx, emp1Actor, c2.ID))
	mustTransition(wf.StartReview(ctx, reviewerActor, c2.ID))

	// Claim 3: rejected.
	c3 := models.Claim{
		EmployeeID: emp2.ID, BeneficiaryType: models.BeneficiarySelf, ServiceTypeID: pharmacy.ID,
		PlanID: *emp2.PlanID, RequestedAmount: 200000, ReceiptDate: time.Now().AddDate(0, 0, -15),
		Description: "خرید دارو - بدون نسخه", Status: models.ClaimDraft, CreatedBy: empUser2.ID,
	}
	gdb.Create(&c3)
	mustTransition(wf.Submit(ctx, emp2Actor, c3.ID))
	mustTransition(wf.StartReview(ctx, reviewerActor, c3.ID))
	mustTransition(wf.Reject(ctx, reviewerActor, c3.ID, "نسخه پزشک پیوست نشده است"))

	// Claim 4: freshly submitted, untouched — for the reviewer queue demo.
	c4 := models.Claim{
		EmployeeID: emp2.ID, BeneficiaryType: models.BeneficiarySelf, ServiceTypeID: outpatient.ID,
		PlanID: *emp2.PlanID, RequestedAmount: 500000, ReceiptDate: time.Now(),
		Description: "ویزیت سرپایی - پیگیری", Status: models.ClaimDraft, CreatedBy: empUser2.ID,
	}
	gdb.Create(&c4)
	mustTransition(wf.Submit(ctx, emp2Actor, c4.ID))
}

func mustTransition(claim *models.Claim, err error) {
	if err != nil {
		log.Fatalf("seed transition failed: %v", err)
	}
}
