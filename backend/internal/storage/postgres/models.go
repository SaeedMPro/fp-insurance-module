// Package postgres is the persistence layer: GORM row types (column mapping
// only — never JSON), mappers to/from domain entities, and the Store that
// implements every repository interface the services declare. GORM does not
// appear anywhere outside this package.
//
// Schema evolution belongs to backend/migrations (golang-migrate); these row
// types only describe how to read and write the already-migrated schema.
package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type contractRow struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string
	StartDate time.Time
	EndDate   time.Time
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (contractRow) TableName() string { return "insurance_contracts" }

type planRow struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ContractID  uuid.UUID `gorm:"type:uuid"`
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (planRow) TableName() string { return "coverage_plans" }

type employeeRow struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonnelNo      string
	FullName         string
	NationalID       string
	EmploymentStatus string
	HireDate         time.Time
	Department       string
	PlanID           *uuid.UUID `gorm:"type:uuid"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (employeeRow) TableName() string { return "employees" }

type userRow struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username     string
	PasswordHash string
	FullName     string
	Role         string
	EmployeeID   *uuid.UUID `gorm:"type:uuid"`
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userRow) TableName() string { return "users" }

type serviceTypeRow struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code      string
	Name      string
	NameFa    string
	CreatedAt time.Time
}

func (serviceTypeRow) TableName() string { return "service_types" }

type ruleRow struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PlanID            uuid.UUID `gorm:"type:uuid"`
	ServiceTypeID     uuid.UUID `gorm:"type:uuid"`
	CoveragePercent   float64
	PerClaimCap       *float64
	AnnualCap         *float64
	WaitingPeriodDays int
	EligibleRelations pq.StringArray `gorm:"type:text[]"`
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time
	CreatedBy         *uuid.UUID `gorm:"type:uuid"`
	CreatedAt         time.Time
}

func (ruleRow) TableName() string { return "coverage_rules" }

type dependentRow struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EmployeeID uuid.UUID `gorm:"type:uuid"`
	FullName   string
	Relation   string
	NationalID string
	BirthDate  *time.Time
	CreatedAt  time.Time
}

func (dependentRow) TableName() string { return "dependents" }

type claimRow struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EmployeeID             uuid.UUID `gorm:"type:uuid"`
	BeneficiaryType        string
	DependentID            *uuid.UUID `gorm:"type:uuid"`
	ServiceTypeID          uuid.UUID  `gorm:"type:uuid"`
	PlanID                 uuid.UUID  `gorm:"type:uuid"`
	RequestedAmount        float64
	ReceiptDate            time.Time
	Description            string
	Status                 string
	CoveragePercentApplied *float64
	PayableAmount          *float64
	RejectionReason        string
	SubmittedAt            *time.Time
	ReviewedBy             *uuid.UUID `gorm:"type:uuid"`
	ReviewedAt             *time.Time
	PaidAt                 *time.Time
	ClosedAt               *time.Time
	CreatedBy              uuid.UUID `gorm:"type:uuid"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (claimRow) TableName() string { return "claims" }

type paymentRow struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClaimID          uuid.UUID `gorm:"type:uuid"`
	Amount           float64
	PaymentReference string
	Status           string
	PaidAt           time.Time
}

func (paymentRow) TableName() string { return "payments" }

type auditRow struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EntityType    string
	EntityID      string
	Action        string
	ActorUserID   *uuid.UUID `gorm:"type:uuid"`
	ActorUsername string
	BeforeData    jsonMap `gorm:"type:jsonb"`
	AfterData     jsonMap `gorm:"type:jsonb"`
	Metadata      jsonMap `gorm:"type:jsonb"`
	OccurredAt    time.Time
}

func (auditRow) TableName() string { return "audit_logs" }

type apiKeyRow struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name       string
	APIKeyHash string
	IsActive   bool
	CreatedAt  time.Time
}

func (apiKeyRow) TableName() string { return "integration_api_keys" }
