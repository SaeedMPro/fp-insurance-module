package postgres

import "insurance-module/internal/domain"

// Row ↔ domain mappers. Deliberately mechanical: no logic beyond type
// conversion lives here.

func (r contractRow) toDomain() domain.InsuranceContract {
	return domain.InsuranceContract{
		ID: r.ID, Name: r.Name, StartDate: r.StartDate, EndDate: r.EndDate,
		IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func contractFromDomain(c domain.InsuranceContract) contractRow {
	return contractRow{
		ID: c.ID, Name: c.Name, StartDate: c.StartDate, EndDate: c.EndDate,
		IsActive: c.IsActive, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func (r planRow) toDomain() domain.CoveragePlan {
	return domain.CoveragePlan{
		ID: r.ID, ContractID: r.ContractID, Name: r.Name, Description: r.Description,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func planFromDomain(p domain.CoveragePlan) planRow {
	return planRow{
		ID: p.ID, ContractID: p.ContractID, Name: p.Name, Description: p.Description,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func (r employeeRow) toDomain() domain.Employee {
	return domain.Employee{
		ID: r.ID, PersonnelNo: r.PersonnelNo, FullName: r.FullName, NationalID: r.NationalID,
		EmploymentStatus: domain.EmploymentStatus(r.EmploymentStatus), HireDate: r.HireDate,
		Department: r.Department, PlanID: r.PlanID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func employeeFromDomain(e domain.Employee) employeeRow {
	return employeeRow{
		ID: e.ID, PersonnelNo: e.PersonnelNo, FullName: e.FullName, NationalID: e.NationalID,
		EmploymentStatus: string(e.EmploymentStatus), HireDate: e.HireDate,
		Department: e.Department, PlanID: e.PlanID, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

func (r userRow) toDomain() domain.User {
	return domain.User{
		ID: r.ID, Username: r.Username, PasswordHash: r.PasswordHash, FullName: r.FullName,
		Role: domain.Role(r.Role), EmployeeID: r.EmployeeID, IsActive: r.IsActive,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func userFromDomain(u domain.User) userRow {
	return userRow{
		ID: u.ID, Username: u.Username, PasswordHash: u.PasswordHash, FullName: u.FullName,
		Role: string(u.Role), EmployeeID: u.EmployeeID, IsActive: u.IsActive,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func (r serviceTypeRow) toDomain() domain.ServiceType {
	return domain.ServiceType{ID: r.ID, Code: r.Code, Name: r.Name, NameFa: r.NameFa, CreatedAt: r.CreatedAt}
}

func (r ruleRow) toDomain() domain.CoverageRule {
	relations := make([]domain.Relation, 0, len(r.EligibleRelations))
	for _, rel := range r.EligibleRelations {
		relations = append(relations, domain.Relation(rel))
	}
	return domain.CoverageRule{
		ID: r.ID, PlanID: r.PlanID, ServiceTypeID: r.ServiceTypeID,
		CoveragePercent:   domain.PercentFromFloat(r.CoveragePercent),
		PerClaimCap:       domain.RialPtrFromFloatPtr(r.PerClaimCap),
		AnnualCap:         domain.RialPtrFromFloatPtr(r.AnnualCap),
		WaitingPeriodDays: r.WaitingPeriodDays, EligibleRelations: relations,
		EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt,
	}
}

func ruleFromDomain(c domain.CoverageRule) ruleRow {
	relations := make([]string, 0, len(c.EligibleRelations))
	for _, rel := range c.EligibleRelations {
		relations = append(relations, string(rel))
	}
	return ruleRow{
		ID: c.ID, PlanID: c.PlanID, ServiceTypeID: c.ServiceTypeID,
		CoveragePercent:   c.CoveragePercent.Float(),
		PerClaimCap:       domain.FloatPtrFromRialPtr(c.PerClaimCap),
		AnnualCap:         domain.FloatPtrFromRialPtr(c.AnnualCap),
		WaitingPeriodDays: c.WaitingPeriodDays, EligibleRelations: relations,
		EffectiveFrom: c.EffectiveFrom, EffectiveTo: c.EffectiveTo,
		CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt,
	}
}

func (r dependentRow) toDomain() domain.Dependent {
	return domain.Dependent{
		ID: r.ID, EmployeeID: r.EmployeeID, FullName: r.FullName,
		Relation: domain.Relation(r.Relation), NationalID: r.NationalID,
		BirthDate: r.BirthDate, CreatedAt: r.CreatedAt,
	}
}

func dependentFromDomain(d domain.Dependent) dependentRow {
	return dependentRow{
		ID: d.ID, EmployeeID: d.EmployeeID, FullName: d.FullName,
		Relation: string(d.Relation), NationalID: d.NationalID,
		BirthDate: d.BirthDate, CreatedAt: d.CreatedAt,
	}
}

func (r claimRow) toDomain() domain.Claim {
	return domain.Claim{
		ID: r.ID, EmployeeID: r.EmployeeID,
		BeneficiaryType: domain.BeneficiaryType(r.BeneficiaryType), DependentID: r.DependentID,
		ServiceTypeID: r.ServiceTypeID, PlanID: r.PlanID,
		RequestedAmount: domain.RialFromFloat(r.RequestedAmount),
		ReceiptDate:     r.ReceiptDate, Description: r.Description,
		Status:                 domain.ClaimStatus(r.Status),
		CoveragePercentApplied: percentPtrFromFloatPtr(r.CoveragePercentApplied),
		PayableAmount:          domain.RialPtrFromFloatPtr(r.PayableAmount),
		RejectionReason:        r.RejectionReason,
		SubmittedAt:            r.SubmittedAt, ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt,
		PaidAt: r.PaidAt, ClosedAt: r.ClosedAt,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func claimFromDomain(c domain.Claim) claimRow {
	return claimRow{
		ID: c.ID, EmployeeID: c.EmployeeID,
		BeneficiaryType: string(c.BeneficiaryType), DependentID: c.DependentID,
		ServiceTypeID: c.ServiceTypeID, PlanID: c.PlanID,
		RequestedAmount: c.RequestedAmount.Float(),
		ReceiptDate:     c.ReceiptDate, Description: c.Description,
		Status:                 string(c.Status),
		CoveragePercentApplied: floatPtrFromPercentPtr(c.CoveragePercentApplied),
		PayableAmount:          domain.FloatPtrFromRialPtr(c.PayableAmount),
		RejectionReason:        c.RejectionReason,
		SubmittedAt:            c.SubmittedAt, ReviewedBy: c.ReviewedBy, ReviewedAt: c.ReviewedAt,
		PaidAt: c.PaidAt, ClosedAt: c.ClosedAt,
		CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func paymentFromDomain(p domain.Payment) paymentRow {
	return paymentRow{
		ID: p.ID, ClaimID: p.ClaimID, Amount: p.Amount.Float(),
		PaymentReference: p.PaymentReference, Status: string(p.Status), PaidAt: p.PaidAt,
	}
}

func (r auditRow) toDomain() domain.AuditLog {
	return domain.AuditLog{
		ID: r.ID, EntityType: r.EntityType, EntityID: r.EntityID, Action: r.Action,
		ActorUserID: r.ActorUserID, ActorUsername: r.ActorUsername,
		BeforeData: r.BeforeData, AfterData: r.AfterData, Metadata: r.Metadata,
		OccurredAt: r.OccurredAt,
	}
}

func auditFromDomain(a domain.AuditLog) auditRow {
	return auditRow{
		ID: a.ID, EntityType: a.EntityType, EntityID: a.EntityID, Action: a.Action,
		ActorUserID: a.ActorUserID, ActorUsername: a.ActorUsername,
		BeforeData: a.BeforeData, AfterData: a.AfterData, Metadata: a.Metadata,
		OccurredAt: a.OccurredAt,
	}
}

// percentPtrFromFloatPtr / floatPtrFromPercentPtr map the nullable
// coverage_percent_applied column (NUMERIC(5,2)) to basis points and back.
func percentPtrFromFloatPtr(f *float64) *domain.Percent {
	if f == nil {
		return nil
	}
	p := domain.PercentFromFloat(*f)
	return &p
}

func floatPtrFromPercentPtr(p *domain.Percent) *float64 {
	if p == nil {
		return nil
	}
	f := p.Float()
	return &f
}
