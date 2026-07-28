package domain

import (
	"time"

	"github.com/google/uuid"
)

// Query/filter contracts shared between services and their repositories, and
// the read-model result types for reporting. Kept in domain so the service
// and storage layers share one vocabulary without importing each other.

// Page is a normalized pagination request. NewPage clamps to sane bounds.
type Page struct {
	Number int
	Size   int
}

func NewPage(number, size int) Page {
	if number < 1 {
		number = 1
	}
	if size < 1 || size > 200 {
		size = 50
	}
	return Page{Number: number, Size: size}
}

func (p Page) Offset() int { return (p.Number - 1) * p.Size }

type ClaimFilter struct {
	CreatedBy     *uuid.UUID // employees see only their own claims
	EmployeeID    string
	Status        ClaimStatus
	ServiceTypeID string
	From, To      *time.Time
	Page          Page
}

type RuleFilter struct {
	PlanID        string
	ServiceTypeID string
}

type AuditFilter struct {
	EntityType  string
	EntityID    string
	ActorUserID *uuid.UUID
	Action      string
	From, To    *time.Time
	Page        Page
}

type EmployeeFilter struct {
	Query string // matches full name / personnel number
	Page  Page
}

type ReportRange struct {
	From, To *time.Time
}

// ---- reporting read models ----

type ReportSummary struct {
	TotalClaims             int64
	TotalPaidAmount         Rial
	PendingReview           int64
	ApprovedAwaitingPayment int64
	Rejected                int64
}

type EmployeeSpend struct {
	EmployeeID   uuid.UUID
	EmployeeName string
	PersonnelNo  string
	TotalPaid    Rial
	ClaimCount   int64
}

type ServiceTypeSpend struct {
	ServiceTypeCode string
	ServiceTypeName string
	TotalPaid       Rial
	ClaimCount      int64
}

type MonthSpend struct {
	Month      string
	TotalPaid  Rial
	ClaimCount int64
}

// RemainingCap summarises one service type's benefit for an employee: the
// active rule's terms plus how much of the annual cap is already used.
type RemainingCap struct {
	ServiceTypeCode string
	ServiceTypeName string
	CoveragePercent Percent
	PerClaimCap     *Rial
	AnnualCap       *Rial
	UsedAnnual      Rial
	RemainingAnnual *Rial
}
