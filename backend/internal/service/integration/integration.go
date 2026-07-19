// Package integration is the parent-system seam: API-key verification,
// employee master-data sync (bulk upsert by personnel number), and claim
// status lookup. A live HR connection is explicitly out of scope — this is
// the boundary a real one would call.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

type Repo interface {
	ActiveAPIKeyExists(ctx context.Context, keyHash string) (bool, error)
	GetEmployeeByPersonnelNo(ctx context.Context, personnelNo string) (domain.Employee, bool, error)
	CreateEmployee(ctx context.Context, e *domain.Employee) error
	SaveEmployee(ctx context.Context, e *domain.Employee) error
	GetClaim(ctx context.Context, id uuid.UUID) (domain.Claim, error)
}

type Atomic func(ctx context.Context, fn func(Repo) error) error

type Service struct {
	repo   Repo
	atomic Atomic
}

func NewService(repo Repo, atomic Atomic) *Service {
	return &Service{repo: repo, atomic: atomic}
}

// VerifyAPIKey reports whether the presented key matches an active
// integration key (compared by SHA-256 hash; plaintext is never stored).
func (s *Service) VerifyAPIKey(ctx context.Context, key string) (bool, error) {
	sum := sha256.Sum256([]byte(key))
	return s.repo.ActiveAPIKeyExists(ctx, hex.EncodeToString(sum[:]))
}

// SyncEmployee is one employee record as sent by the parent system.
type SyncEmployee struct {
	PersonnelNo      string
	FullName         string
	NationalID       string
	EmploymentStatus domain.EmploymentStatus
	HireDate         time.Time
	Department       string
	PlanID           *uuid.UUID
}

type SyncResult struct {
	Created int
	Updated int
}

// SyncEmployees upserts by personnel_no inside one transaction: either the
// whole batch lands or none of it does.
func (s *Service) SyncEmployees(ctx context.Context, batch []SyncEmployee) (SyncResult, error) {
	var res SyncResult
	err := s.atomic(ctx, func(r Repo) error {
		for _, se := range batch {
			existing, found, err := r.GetEmployeeByPersonnelNo(ctx, se.PersonnelNo)
			if err != nil {
				return err
			}
			if !found {
				emp := domain.Employee{
					PersonnelNo:      se.PersonnelNo,
					FullName:         se.FullName,
					NationalID:       se.NationalID,
					EmploymentStatus: se.EmploymentStatus,
					HireDate:         se.HireDate,
					Department:       se.Department,
					PlanID:           se.PlanID,
				}
				if err := r.CreateEmployee(ctx, &emp); err != nil {
					return err
				}
				res.Created++
				continue
			}
			existing.FullName = se.FullName
			existing.NationalID = se.NationalID
			existing.EmploymentStatus = se.EmploymentStatus
			existing.HireDate = se.HireDate
			existing.Department = se.Department
			if se.PlanID != nil {
				existing.PlanID = se.PlanID
			}
			if err := r.SaveEmployee(ctx, &existing); err != nil {
				return err
			}
			res.Updated++
		}
		return nil
	})
	if err != nil {
		return SyncResult{}, err
	}
	return res, nil
}

// ClaimStatus returns the minimal status view the parent system polls for.
func (s *Service) ClaimStatus(ctx context.Context, id uuid.UUID) (domain.Claim, error) {
	return s.repo.GetClaim(ctx, id)
}
