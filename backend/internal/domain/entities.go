package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entities carry no serialization tags on purpose: the wire format belongs to
// transport/http DTOs and the column mapping to storage/postgres row types.

type InsuranceContract struct {
	ID        uuid.UUID
	Name      string
	StartDate time.Time
	EndDate   time.Time
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CoveragePlan struct {
	ID          uuid.UUID
	ContractID  uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Employee struct {
	ID               uuid.UUID
	PersonnelNo      string
	FullName         string
	NationalID       string
	EmploymentStatus EmploymentStatus
	HireDate         time.Time
	Department       string
	PlanID           *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	FullName     string
	Role         Role
	EmployeeID   *uuid.UUID
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ServiceType struct {
	ID        uuid.UUID
	Code      string
	Name      string
	CreatedAt time.Time
}

// CoverageRule is one config-driven, versioned policy row: everything the
// pricing engine needs for a given (plan, service type) at a point in time.
type CoverageRule struct {
	ID                uuid.UUID
	PlanID            uuid.UUID
	ServiceTypeID     uuid.UUID
	CoveragePercent   Percent
	PerClaimCap       *Rial
	AnnualCap         *Rial
	WaitingPeriodDays int
	EligibleRelations []Relation
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time
	CreatedBy         *uuid.UUID
	CreatedAt         time.Time
}

// EligibleFor reports whether the rule covers services for the given relation.
func (r CoverageRule) EligibleFor(rel Relation) bool {
	for _, v := range r.EligibleRelations {
		if v == rel {
			return true
		}
	}
	return false
}

type Dependent struct {
	ID         uuid.UUID
	EmployeeID uuid.UUID
	FullName   string
	Relation   Relation
	NationalID string
	BirthDate  *time.Time
	CreatedAt  time.Time
}

type Claim struct {
	ID                     uuid.UUID
	EmployeeID             uuid.UUID
	BeneficiaryType        BeneficiaryType
	DependentID            *uuid.UUID
	ServiceTypeID          uuid.UUID
	PlanID                 uuid.UUID
	RequestedAmount        Rial
	ReceiptDate            time.Time
	Description            string
	Status                 ClaimStatus
	CoveragePercentApplied *Percent
	PayableAmount          *Rial
	RejectionReason        string
	SubmittedAt            *time.Time
	ReviewedBy             *uuid.UUID
	ReviewedAt             *time.Time
	PaidAt                 *time.Time
	ClosedAt               *time.Time
	CreatedBy              uuid.UUID
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Payment struct {
	ID               uuid.UUID
	ClaimID          uuid.UUID
	Amount           Rial
	PaymentReference string
	Status           PaymentStatus
	PaidAt           time.Time
}

type AuditLog struct {
	ID            uuid.UUID
	EntityType    string
	EntityID      string
	Action        string
	ActorUserID   *uuid.UUID
	ActorUsername string
	BeforeData    map[string]any
	AfterData     map[string]any
	Metadata      map[string]any
	OccurredAt    time.Time
}

// Actor identifies who performs an action, for both authorization decisions
// and the audit trail. It is derived from the verified JWT (or seed wiring).
type Actor struct {
	UserID   uuid.UUID
	Username string
	Role     Role
}

func (a Actor) IsStaff() bool { return a.Role == RoleAdmin || a.Role == RoleReviewer }
