package api

import (
	"net/http"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"insurance-module/internal/audit"
	"insurance-module/internal/models"
)

func (h *handlers) listServiceTypes(w http.ResponseWriter, r *http.Request) {
	var out []models.ServiceType
	if err := h.d.DB.WithContext(r.Context()).Order("name").Find(&out).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) listContracts(w http.ResponseWriter, r *http.Request) {
	var out []models.InsuranceContract
	if err := h.d.DB.WithContext(r.Context()).Order("start_date DESC").Find(&out).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type createContractRequest struct {
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
}

func (h *handlers) createContract(w http.ResponseWriter, r *http.Request) {
	var req createContractRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c := models.InsuranceContract{Name: req.Name, StartDate: req.StartDate, EndDate: req.EndDate, IsActive: req.IsActive}
	if err := h.d.DB.WithContext(r.Context()).Create(&c).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *handlers) listPlans(w http.ResponseWriter, r *http.Request) {
	q := h.d.DB.WithContext(r.Context()).Order("name")
	if cid := r.URL.Query().Get("contract_id"); cid != "" {
		q = q.Where("contract_id = ?", cid)
	}
	var out []models.CoveragePlan
	if err := q.Find(&out).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type createPlanRequest struct {
	ContractID  string `json:"contract_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *handlers) createPlan(w http.ResponseWriter, r *http.Request) {
	var req createPlanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	contractID, err := parseUUIDField(req.ContractID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid contract_id")
		return
	}
	p := models.CoveragePlan{ContractID: contractID, Name: req.Name, Description: req.Description}
	if err := h.d.DB.WithContext(r.Context()).Create(&p).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *handlers) listCoverageRules(w http.ResponseWriter, r *http.Request) {
	q := h.d.DB.WithContext(r.Context()).Order("effective_from DESC")
	if pid := r.URL.Query().Get("plan_id"); pid != "" {
		q = q.Where("plan_id = ?", pid)
	}
	if sid := r.URL.Query().Get("service_type_id"); sid != "" {
		q = q.Where("service_type_id = ?", sid)
	}
	var out []models.CoverageRule
	if err := q.Find(&out).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type createCoverageRuleRequest struct {
	PlanID            string    `json:"plan_id"`
	ServiceTypeID     string    `json:"service_type_id"`
	CoveragePercent   float64   `json:"coverage_percent"`
	PerClaimCap       *float64  `json:"per_claim_cap"`
	AnnualCap         *float64  `json:"annual_cap"`
	WaitingPeriodDays int       `json:"waiting_period_days"`
	EligibleRelations []string  `json:"eligible_relations"`
	EffectiveFrom     time.Time `json:"effective_from"`
}

// createCoverageRule is the config-driven policy-change endpoint: it closes the
// previous active rule version for the same (plan, service type) and inserts the
// new one, atomically, then records a config_change audit entry — no code
// change or redeploy is ever required to alter benefits.
func (h *handlers) createCoverageRule(w http.ResponseWriter, r *http.Request) {
	var req createCoverageRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	planID, err := parseUUIDField(req.PlanID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid plan_id")
		return
	}
	serviceTypeID, err := parseUUIDField(req.ServiceTypeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service_type_id")
		return
	}

	actor, _ := currentActor(r)
	var created models.CoverageRule
	var previous *models.CoverageRule

	err = h.d.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var prev models.CoverageRule
		err := tx.Where("plan_id = ? AND service_type_id = ? AND effective_to IS NULL", planID, serviceTypeID).
			First(&prev).Error
		if err == nil {
			closeDate := req.EffectiveFrom.AddDate(0, 0, -1)
			// Same-day (or backdated) re-publish: closing with "new - 1 day" would
			// put effective_to before effective_from and violate the row's CHECK
			// constraint. Clamp to the old rule's own start date; on that boundary
			// day the rule engine picks the newest version (created_at tiebreak).
			if closeDate.Before(prev.EffectiveFrom) {
				closeDate = prev.EffectiveFrom
			}
			if err := tx.Model(&prev).Update("effective_to", closeDate).Error; err != nil {
				return err
			}
			previous = &prev
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		created = models.CoverageRule{
			PlanID:            planID,
			ServiceTypeID:     serviceTypeID,
			CoveragePercent:   req.CoveragePercent,
			PerClaimCap:       req.PerClaimCap,
			AnnualCap:         req.AnnualCap,
			WaitingPeriodDays: req.WaitingPeriodDays,
			EligibleRelations: pq.StringArray(req.EligibleRelations),
			EffectiveFrom:     req.EffectiveFrom,
			CreatedBy:         &actor.UserID,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}

		before := map[string]interface{}{}
		if previous != nil {
			before["previous_rule"] = previous
		}
		return h.d.Audit.Log(r.Context(), tx, audit.Entry{
			EntityType:    "coverage_rule",
			EntityID:      created.ID.String(),
			Action:        "config_change",
			ActorUserID:   &actor.UserID,
			ActorUsername: actor.Username,
			Before:        before,
			After:         map[string]interface{}{"new_rule": created},
		})
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}
