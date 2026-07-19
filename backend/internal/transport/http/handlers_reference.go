package http

import (
	"net/http"
	"time"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/coverage"
)

func (s *Server) handleListServiceTypes(w http.ResponseWriter, r *http.Request) {
	items, err := s.coverage.ListServiceTypes(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(items, toServiceTypeDTO))
}

func (s *Server) handleListContracts(w http.ResponseWriter, r *http.Request) {
	items, err := s.coverage.ListContracts(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(items, toContractDTO))
}

type createContractRequest struct {
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
}

func (s *Server) handleCreateContract(w http.ResponseWriter, r *http.Request) {
	var req createContractRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	contract, err := s.coverage.CreateContract(r.Context(), domain.InsuranceContract{
		Name: req.Name, StartDate: req.StartDate, EndDate: req.EndDate, IsActive: req.IsActive,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toContractDTO(contract))
}

func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	items, err := s.coverage.ListPlans(r.Context(), r.URL.Query().Get("contract_id"))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(items, toPlanDTO))
}

type createPlanRequest struct {
	ContractID  string `json:"contract_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req createPlanRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	contractID, err := parseUUID(req.ContractID, "contract_id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	plan, err := s.coverage.CreatePlan(r.Context(), domain.CoveragePlan{
		ContractID: contractID, Name: req.Name, Description: req.Description,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPlanDTO(plan))
}

func (s *Server) handleListCoverageRules(w http.ResponseWriter, r *http.Request) {
	items, err := s.coverage.ListRules(r.Context(), domain.RuleFilter{
		PlanID:        r.URL.Query().Get("plan_id"),
		ServiceTypeID: r.URL.Query().Get("service_type_id"),
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(items, toCoverageRuleDTO))
}

type createCoverageRuleRequest struct {
	PlanID            string            `json:"plan_id"`
	ServiceTypeID     string            `json:"service_type_id"`
	CoveragePercent   float64           `json:"coverage_percent"`
	PerClaimCap       *float64          `json:"per_claim_cap"`
	AnnualCap         *float64          `json:"annual_cap"`
	WaitingPeriodDays int               `json:"waiting_period_days"`
	EligibleRelations []domain.Relation `json:"eligible_relations"`
	EffectiveFrom     time.Time         `json:"effective_from"`
}

func (s *Server) handleCreateCoverageRule(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	var req createCoverageRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	planID, err := parseUUID(req.PlanID, "plan_id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	serviceTypeID, err := parseUUID(req.ServiceTypeID, "service_type_id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	rule, err := s.coverage.PublishRuleVersion(r.Context(), actor, coverage.PublishRuleInput{
		PlanID:            planID,
		ServiceTypeID:     serviceTypeID,
		CoveragePercent:   req.CoveragePercent,
		PerClaimCap:       req.PerClaimCap,
		AnnualCap:         req.AnnualCap,
		WaitingPeriodDays: req.WaitingPeriodDays,
		EligibleRelations: req.EligibleRelations,
		EffectiveFrom:     req.EffectiveFrom,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCoverageRuleDTO(rule))
}
