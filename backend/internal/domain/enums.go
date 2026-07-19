// Package domain holds the pure business types of the supplementary-insurance
// module: entities, enumerations, the error taxonomy, and small primitives
// (Clock). Nothing in this package knows about HTTP, JSON field names, or the
// database — those concerns live in transport/ and storage/ respectively.
package domain

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

type PaymentStatus string

const (
	PaymentSimulated PaymentStatus = "simulated"
	PaymentCompleted PaymentStatus = "completed"
)
