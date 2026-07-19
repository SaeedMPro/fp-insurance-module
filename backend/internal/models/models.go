// Package models defines the GORM entities mapped onto the schema created by
// backend/migrations. GORM AutoMigrate is never used — the SQL migrations are the
// single source of truth for schema; these structs only describe how to read/write it.
// JSON tags on every field double as the wire format returned by the REST API
// (see docs/API-CONTRACT.md), so field names here are the API's field names too.
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleReviewer Role = "reviewer"
	RoleEmployee Role = "employee"
	RoleAuditor  Role = "auditor"
)

type EmploymentStatus string

const (
	EmploymentActive     EmploymentStatus = "active"
	EmploymentTerminated EmploymentStatus = "terminated"
)

type Relation string

const (
	RelationSelf   Relation = "self"
	RelationSpouse Relation = "spouse"
	RelationChild  Relation = "child"
	RelationParent Relation = "parent"
)

type BeneficiaryType string

const (
	BeneficiarySelf      BeneficiaryType = "self"
	BeneficiaryDependent BeneficiaryType = "dependent"
)

type ClaimStatus string

const (
	ClaimDraft             ClaimStatus = "draft"
	ClaimSubmitted         ClaimStatus = "submitted"
	ClaimUnderReview       ClaimStatus = "under_review"
	ClaimReturnedForDocs   ClaimStatus = "returned_for_docs"
	ClaimApproved          ClaimStatus = "approved"
	ClaimRejected          ClaimStatus = "rejected"
	ClaimPaymentCalculated ClaimStatus = "payment_calculated"
	ClaimPaid              ClaimStatus = "paid"
	ClaimClosed            ClaimStatus = "closed"
)

type InsuranceContract struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (InsuranceContract) TableName() string { return "insurance_contracts" }

type CoveragePlan struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ContractID  uuid.UUID `gorm:"type:uuid" json:"contract_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CoveragePlan) TableName() string { return "coverage_plans" }

type Employee struct {
	ID               uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PersonnelNo      string           `json:"personnel_no"`
	FullName         string           `json:"full_name"`
	NationalID       string           `json:"national_id"`
	EmploymentStatus EmploymentStatus `json:"employment_status"`
	HireDate         time.Time        `json:"hire_date"`
	Department       string           `json:"department"`
	PlanID           *uuid.UUID       `gorm:"type:uuid" json:"plan_id"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

func (Employee) TableName() string { return "employees" }

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"full_name"`
	Role         Role       `json:"role"`
	EmployeeID   *uuid.UUID `gorm:"type:uuid" json:"employee_id"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (User) TableName() string { return "users" }

type ServiceType struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	NameFa    string    `json:"name_fa"`
	CreatedAt time.Time `json:"created_at"`
}

func (ServiceType) TableName() string { return "service_types" }

// CoverageRule is one config-driven, versioned policy row: everything the rule
// engine needs to price a claim for a given (plan, service type) at a point in time.
type CoverageRule struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PlanID            uuid.UUID      `gorm:"type:uuid" json:"plan_id"`
	ServiceTypeID     uuid.UUID      `gorm:"type:uuid" json:"service_type_id"`
	CoveragePercent   float64        `json:"coverage_percent"`
	PerClaimCap       *float64       `json:"per_claim_cap"`
	AnnualCap         *float64       `json:"annual_cap"`
	WaitingPeriodDays int            `json:"waiting_period_days"`
	EligibleRelations pq.StringArray `gorm:"type:text[]" json:"eligible_relations"`
	EffectiveFrom     time.Time      `json:"effective_from"`
	EffectiveTo       *time.Time     `json:"effective_to"`
	CreatedBy         *uuid.UUID     `gorm:"type:uuid" json:"created_by"`
	CreatedAt         time.Time      `json:"created_at"`
}

func (CoverageRule) TableName() string { return "coverage_rules" }

type Dependent struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EmployeeID uuid.UUID  `gorm:"type:uuid" json:"employee_id"`
	FullName   string     `json:"full_name"`
	Relation   Relation   `json:"relation"`
	NationalID string     `json:"national_id"`
	BirthDate  *time.Time `json:"birth_date"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (Dependent) TableName() string { return "dependents" }

type Claim struct {
	ID                     uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EmployeeID             uuid.UUID       `gorm:"type:uuid" json:"employee_id"`
	BeneficiaryType        BeneficiaryType `json:"beneficiary_type"`
	DependentID            *uuid.UUID      `gorm:"type:uuid" json:"dependent_id"`
	ServiceTypeID          uuid.UUID       `gorm:"type:uuid" json:"service_type_id"`
	PlanID                 uuid.UUID       `gorm:"type:uuid" json:"plan_id"`
	RequestedAmount        float64         `json:"requested_amount"`
	ReceiptDate            time.Time       `json:"receipt_date"`
	Description            string          `json:"description"`
	Status                 ClaimStatus     `json:"status"`
	CoveragePercentApplied *float64        `json:"coverage_percent_applied"`
	PayableAmount          *float64        `json:"payable_amount"`
	RejectionReason        string          `json:"rejection_reason,omitempty"`
	SubmittedAt            *time.Time      `json:"submitted_at"`
	ReviewedBy             *uuid.UUID      `gorm:"type:uuid" json:"reviewed_by"`
	ReviewedAt             *time.Time      `json:"reviewed_at"`
	PaidAt                 *time.Time      `json:"paid_at"`
	ClosedAt               *time.Time      `json:"closed_at"`
	CreatedBy              uuid.UUID       `gorm:"type:uuid" json:"created_by"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

func (Claim) TableName() string { return "claims" }

type ClaimAttachment struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClaimID    uuid.UUID `gorm:"type:uuid" json:"claim_id"`
	FileName   string    `json:"file_name"`
	FilePath   string    `json:"file_path"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func (ClaimAttachment) TableName() string { return "claim_attachments" }

type PaymentStatus string

const (
	PaymentSimulated PaymentStatus = "simulated"
	PaymentCompleted PaymentStatus = "completed"
)

type Payment struct {
	ID               uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClaimID          uuid.UUID     `gorm:"type:uuid" json:"claim_id"`
	Amount           float64       `json:"amount"`
	PaymentReference string        `json:"payment_reference"`
	Status           PaymentStatus `json:"status"`
	PaidAt           time.Time     `json:"paid_at"`
}

func (Payment) TableName() string { return "payments" }

type AuditLog struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EntityType    string     `json:"entity_type"`
	EntityID      string     `json:"entity_id"`
	Action        string     `json:"action"`
	ActorUserID   *uuid.UUID `gorm:"type:uuid" json:"actor_user_id"`
	ActorUsername string     `json:"actor_username"`
	BeforeData    JSONMap    `gorm:"type:jsonb" json:"before_data"`
	AfterData     JSONMap    `gorm:"type:jsonb" json:"after_data"`
	Metadata      JSONMap    `gorm:"type:jsonb" json:"metadata"`
	OccurredAt    time.Time  `json:"occurred_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

type IntegrationAPIKey struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name       string    `json:"name"`
	APIKeyHash string    `json:"-"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

func (IntegrationAPIKey) TableName() string { return "integration_api_keys" }
