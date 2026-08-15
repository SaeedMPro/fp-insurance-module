package http

import (
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

// Response DTOs. These define the public wire format — field names here are
// the API contract (docs/API-CONTRACT.md) and must not drift with the schema.

type userDTO struct {
	ID         uuid.UUID   `json:"id"`
	Username   string      `json:"username"`
	FullName   string      `json:"full_name"`
	Role       domain.Role `json:"role"`
	EmployeeID *uuid.UUID  `json:"employee_id"`
	IsActive   bool        `json:"is_active"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

func toUserDTO(u domain.User) userDTO {
	return userDTO{
		ID: u.ID, Username: u.Username, FullName: u.FullName, Role: u.Role,
		EmployeeID: u.EmployeeID, IsActive: u.IsActive, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

type employeeDTO struct {
	ID               uuid.UUID               `json:"id"`
	PersonnelNo      string                  `json:"personnel_no"`
	FullName         string                  `json:"full_name"`
	NationalID       string                  `json:"national_id"`
	EmploymentStatus domain.EmploymentStatus `json:"employment_status"`
	HireDate         time.Time               `json:"hire_date"`
	Department       string                  `json:"department"`
	PlanID           *uuid.UUID              `json:"plan_id"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

func toEmployeeDTO(e domain.Employee) employeeDTO {
	return employeeDTO{
		ID: e.ID, PersonnelNo: e.PersonnelNo, FullName: e.FullName, NationalID: e.NationalID,
		EmploymentStatus: e.EmploymentStatus, HireDate: e.HireDate, Department: e.Department,
		PlanID: e.PlanID, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

type dependentDTO struct {
	ID         uuid.UUID       `json:"id"`
	EmployeeID uuid.UUID       `json:"employee_id"`
	FullName   string          `json:"full_name"`
	Relation   domain.Relation `json:"relation"`
	NationalID string          `json:"national_id"`
	BirthDate  *time.Time      `json:"birth_date"`
	CreatedAt  time.Time       `json:"created_at"`
}

func toDependentDTO(d domain.Dependent) dependentDTO {
	return dependentDTO{
		ID: d.ID, EmployeeID: d.EmployeeID, FullName: d.FullName, Relation: d.Relation,
		NationalID: d.NationalID, BirthDate: d.BirthDate, CreatedAt: d.CreatedAt,
	}
}

type serviceTypeDTO struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func toServiceTypeDTO(s domain.ServiceType) serviceTypeDTO {
	return serviceTypeDTO{ID: s.ID, Code: s.Code, Name: s.Name, CreatedAt: s.CreatedAt}
}

type contractDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toContractDTO(c domain.InsuranceContract) contractDTO {
	return contractDTO{
		ID: c.ID, Name: c.Name, StartDate: c.StartDate, EndDate: c.EndDate,
		IsActive: c.IsActive, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

type planDTO struct {
	ID          uuid.UUID `json:"id"`
	ContractID  uuid.UUID `json:"contract_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toPlanDTO(p domain.CoveragePlan) planDTO {
	return planDTO{
		ID: p.ID, ContractID: p.ContractID, Name: p.Name, Description: p.Description,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

type coverageRuleDTO struct {
	ID                uuid.UUID         `json:"id"`
	PlanID            uuid.UUID         `json:"plan_id"`
	ServiceTypeID     uuid.UUID         `json:"service_type_id"`
	CoveragePercent   float64           `json:"coverage_percent"`
	PerClaimCap       *float64          `json:"per_claim_cap"`
	AnnualCap         *float64          `json:"annual_cap"`
	WaitingPeriodDays int               `json:"waiting_period_days"`
	EligibleRelations []domain.Relation `json:"eligible_relations"`
	EffectiveFrom     time.Time         `json:"effective_from"`
	EffectiveTo       *time.Time        `json:"effective_to"`
	CreatedBy         *uuid.UUID        `json:"created_by"`
	CreatedAt         time.Time         `json:"created_at"`
}

func toCoverageRuleDTO(r domain.CoverageRule) coverageRuleDTO {
	return coverageRuleDTO{
		ID: r.ID, PlanID: r.PlanID, ServiceTypeID: r.ServiceTypeID,
		CoveragePercent:   r.CoveragePercent.Float(),
		PerClaimCap:       domain.FloatPtrFromRialPtr(r.PerClaimCap),
		AnnualCap:         domain.FloatPtrFromRialPtr(r.AnnualCap),
		WaitingPeriodDays: r.WaitingPeriodDays, EligibleRelations: r.EligibleRelations,
		EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt,
	}
}

type claimDTO struct {
	ID                     uuid.UUID              `json:"id"`
	EmployeeID             uuid.UUID              `json:"employee_id"`
	BeneficiaryType        domain.BeneficiaryType `json:"beneficiary_type"`
	DependentID            *uuid.UUID             `json:"dependent_id"`
	ServiceTypeID          uuid.UUID              `json:"service_type_id"`
	PlanID                 uuid.UUID              `json:"plan_id"`
	RequestedAmount        float64                `json:"requested_amount"`
	ReceiptDate            time.Time              `json:"receipt_date"`
	Description            string                 `json:"description"`
	Status                 domain.ClaimStatus     `json:"status"`
	CoveragePercentApplied *float64               `json:"coverage_percent_applied"`
	PayableAmount          *float64               `json:"payable_amount"`
	RejectionReason        string                 `json:"rejection_reason,omitempty"`
	SubmittedAt            *time.Time             `json:"submitted_at"`
	ReviewedBy             *uuid.UUID             `json:"reviewed_by"`
	ReviewedAt             *time.Time             `json:"reviewed_at"`
	PaidAt                 *time.Time             `json:"paid_at"`
	ClosedAt               *time.Time             `json:"closed_at"`
	CreatedBy              uuid.UUID              `json:"created_by"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`
}

func toClaimDTO(c domain.Claim) claimDTO {
	return claimDTO{
		ID: c.ID, EmployeeID: c.EmployeeID, BeneficiaryType: c.BeneficiaryType,
		DependentID: c.DependentID, ServiceTypeID: c.ServiceTypeID, PlanID: c.PlanID,
		RequestedAmount: c.RequestedAmount.Float(),
		ReceiptDate:     c.ReceiptDate, Description: c.Description,
		Status:                 c.Status,
		CoveragePercentApplied: percentFloatPtr(c.CoveragePercentApplied),
		PayableAmount:          domain.FloatPtrFromRialPtr(c.PayableAmount),
		RejectionReason:        c.RejectionReason,
		SubmittedAt:            c.SubmittedAt, ReviewedBy: c.ReviewedBy, ReviewedAt: c.ReviewedAt,
		PaidAt: c.PaidAt, ClosedAt: c.ClosedAt,
		CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

type auditLogDTO struct {
	ID            uuid.UUID      `json:"id"`
	EntityType    string         `json:"entity_type"`
	EntityID      string         `json:"entity_id"`
	Action        string         `json:"action"`
	ActorUserID   *uuid.UUID     `json:"actor_user_id"`
	ActorUsername string         `json:"actor_username"`
	BeforeData    map[string]any `json:"before_data"`
	AfterData     map[string]any `json:"after_data"`
	Metadata      map[string]any `json:"metadata"`
	OccurredAt    time.Time      `json:"occurred_at"`
}

func toAuditLogDTO(a domain.AuditLog) auditLogDTO {
	return auditLogDTO{
		ID: a.ID, EntityType: a.EntityType, EntityID: a.EntityID, Action: a.Action,
		ActorUserID: a.ActorUserID, ActorUsername: a.ActorUsername,
		BeforeData: a.BeforeData, AfterData: a.AfterData, Metadata: a.Metadata,
		OccurredAt: a.OccurredAt,
	}
}

type remainingCapDTO struct {
	ServiceTypeCode string   `json:"service_type_code"`
	ServiceTypeName string   `json:"service_type_name"`
	CoveragePercent float64  `json:"coverage_percent"`
	PerClaimCap     *float64 `json:"per_claim_cap"`
	AnnualCap       *float64 `json:"annual_cap"`
	UsedAnnual      float64  `json:"used_annual"`
	RemainingAnnual *float64 `json:"remaining_annual"`
}

func toRemainingCapDTO(c domain.RemainingCap) remainingCapDTO {
	return remainingCapDTO{
		ServiceTypeCode: c.ServiceTypeCode, ServiceTypeName: c.ServiceTypeName,
		CoveragePercent: c.CoveragePercent.Float(),
		PerClaimCap:     domain.FloatPtrFromRialPtr(c.PerClaimCap),
		AnnualCap:       domain.FloatPtrFromRialPtr(c.AnnualCap),
		UsedAnnual:      c.UsedAnnual.Float(),
		RemainingAnnual: domain.FloatPtrFromRialPtr(c.RemainingAnnual),
	}
}

type summaryDTO struct {
	TotalClaims             int64   `json:"total_claims"`
	TotalPaidAmount         float64 `json:"total_paid_amount"`
	PendingReview           int64   `json:"pending_review"`
	ApprovedAwaitingPayment int64   `json:"approved_awaiting_payment"`
	Rejected                int64   `json:"rejected"`
}

type employeeSpendDTO struct {
	EmployeeID   uuid.UUID `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	PersonnelNo  string    `json:"personnel_no"`
	TotalPaid    float64   `json:"total_paid"`
	ClaimCount   int64     `json:"claim_count"`
}

type serviceTypeSpendDTO struct {
	ServiceTypeCode string  `json:"service_type_code"`
	ServiceTypeName string  `json:"service_type_name"`
	TotalPaid       float64 `json:"total_paid"`
	ClaimCount      int64   `json:"claim_count"`
}

type monthSpendDTO struct {
	Month      string  `json:"month"`
	TotalPaid  float64 `json:"total_paid"`
	ClaimCount int64   `json:"claim_count"`
}

// listDTO is the {items, total} envelope every paginated list uses.
type listDTO[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

func mapSlice[In any, Out any](in []In, f func(In) Out) []Out {
	out := make([]Out, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}
	return out
}

// percentFloatPtr renders an optional basis-point percentage as the JSON
// number the contract specifies.
func percentFloatPtr(p *domain.Percent) *float64 {
	if p == nil {
		return nil
	}
	f := p.Float()
	return &f
}
