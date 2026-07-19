package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"insurance-module/internal/models"
	"insurance-module/internal/workflow"
)

type createClaimRequest struct {
	EmployeeID      *string                `json:"employee_id"`
	BeneficiaryType models.BeneficiaryType `json:"beneficiary_type"`
	DependentID     *string                `json:"dependent_id"`
	ServiceTypeID   string                 `json:"service_type_id"`
	RequestedAmount float64                `json:"requested_amount"`
	ReceiptDate     time.Time              `json:"receipt_date"`
	Description     string                 `json:"description"`
}

func (h *handlers) createClaim(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentActor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if actor.Role != models.RoleEmployee && actor.Role != models.RoleAdmin {
		writeError(w, http.StatusForbidden, "only employees or admins may submit claims")
		return
	}

	var req createClaimRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	employeeID, err := h.resolveClaimEmployeeID(r, actor, req.EmployeeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var employee models.Employee
	if err := h.d.DB.WithContext(r.Context()).First(&employee, "id = ?", employeeID).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	if employee.PlanID == nil {
		writeError(w, http.StatusUnprocessableEntity, "employee has no coverage plan assigned")
		return
	}

	serviceTypeID, err := parseUUIDField(req.ServiceTypeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service_type_id")
		return
	}

	claim := models.Claim{
		EmployeeID:      employeeID,
		BeneficiaryType: req.BeneficiaryType,
		ServiceTypeID:   serviceTypeID,
		PlanID:          *employee.PlanID,
		RequestedAmount: req.RequestedAmount,
		ReceiptDate:     req.ReceiptDate,
		Description:     req.Description,
		Status:          models.ClaimDraft,
		CreatedBy:       actor.UserID,
	}
	if req.BeneficiaryType == models.BeneficiaryDependent && req.DependentID != nil {
		depID, err := parseUUIDField(*req.DependentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dependent_id")
			return
		}
		claim.DependentID = &depID
	}

	if err := h.d.DB.WithContext(r.Context()).Create(&claim).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, claim)
}

// resolveClaimEmployeeID enforces "employee forces their own employee_id, admin may
// specify any employee_id" from docs/API-CONTRACT.md.
func (h *handlers) resolveClaimEmployeeID(r *http.Request, actor workflow.Actor, requested *string) (uuid.UUID, error) {
	if actor.Role == models.RoleAdmin {
		if requested == nil {
			return uuid.UUID{}, errors.New("employee_id is required")
		}
		return parseUUIDField(*requested)
	}

	var user models.User
	if err := h.d.DB.WithContext(r.Context()).First(&user, "id = ?", actor.UserID).Error; err != nil {
		return uuid.UUID{}, errors.New("could not resolve caller's employee record")
	}
	if user.EmployeeID == nil {
		return uuid.UUID{}, errors.New("this account is not linked to an employee record")
	}
	return *user.EmployeeID, nil
}

type claimListResponse struct {
	Items []models.Claim `json:"items"`
	Total int64          `json:"total"`
}

func (h *handlers) listClaims(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentActor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	page, pageSize := pagination(r)
	q := h.d.DB.WithContext(r.Context()).Model(&models.Claim{})

	if actor.Role == models.RoleEmployee {
		q = q.Where("created_by = ?", actor.UserID)
	} else if eid := r.URL.Query().Get("employee_id"); eid != "" {
		q = q.Where("employee_id = ?", eid)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if sid := r.URL.Query().Get("service_type_id"); sid != "" {
		q = q.Where("service_type_id = ?", sid)
	}
	from, to := dateRange(r)
	if from != nil {
		q = q.Where("receipt_date >= ?", *from)
	}
	if to != nil {
		q = q.Where("receipt_date <= ?", *to)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	var items []models.Claim
	if err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, claimListResponse{Items: items, Total: total})
}

func (h *handlers) loadClaimForAccess(w http.ResponseWriter, r *http.Request) (*models.Claim, bool) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return nil, false
	}
	var claim models.Claim
	if err := h.d.DB.WithContext(r.Context()).First(&claim, "id = ?", id).Error; err != nil {
		mapDomainError(w, err)
		return nil, false
	}
	if !h.authorizeClaimAccess(r, &claim) {
		writeError(w, http.StatusForbidden, "not permitted to access this claim")
		return nil, false
	}
	return &claim, true
}

func (h *handlers) getClaim(w http.ResponseWriter, r *http.Request) {
	claim, ok := h.loadClaimForAccess(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, claim)
}

func (h *handlers) claimHistory(w http.ResponseWriter, r *http.Request) {
	claim, ok := h.loadClaimForAccess(w, r)
	if !ok {
		return
	}
	trail, err := h.d.Audit.Trail(r.Context(), "claim", claim.ID.String())
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trail)
}

type reasonRequest struct {
	Reason string `json:"reason"`
}

func (h *handlers) claimSubmit(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error) {
		return h.d.Workflow.Submit(r.Context(), actor, id)
	})
}

func (h *handlers) claimResubmit(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error) {
		return h.d.Workflow.Resubmit(r.Context(), actor, id)
	})
}

func (h *handlers) claimStartReview(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error) {
		return h.d.Workflow.StartReview(r.Context(), actor, id)
	})
}

func (h *handlers) claimApprove(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error) {
		return h.d.Workflow.Approve(r.Context(), actor, id)
	})
}

func (h *handlers) claimMarkPaid(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error) {
		return h.d.Workflow.MarkPaid(r.Context(), actor, id)
	})
}

func (h *handlers) claimClose(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error) {
		return h.d.Workflow.Close(r.Context(), actor, id)
	})
}

func (h *handlers) claimReject(w http.ResponseWriter, r *http.Request) {
	var req reasonRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	h.runTransition(w, r, func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error) {
		return h.d.Workflow.Reject(r.Context(), actor, id, req.Reason)
	})
}

func (h *handlers) claimReturnForDocs(w http.ResponseWriter, r *http.Request) {
	var req reasonRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	h.runTransition(w, r, func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error) {
		return h.d.Workflow.ReturnForDocs(r.Context(), actor, id, req.Reason)
	})
}

// runTransition is the common glue for every claim-lifecycle endpoint: resolve
// the actor and claim id, invoke the workflow engine action, and map its result
// (or one of the workflow/ruleengine sentinel errors) to an HTTP response.
func (h *handlers) runTransition(w http.ResponseWriter, r *http.Request, action func(actor workflow.Actor, id uuid.UUID) (*models.Claim, error)) {
	actor, ok := currentActor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	claim, err := action(actor, id)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, claim)
}
