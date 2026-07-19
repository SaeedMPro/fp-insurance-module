// Package api assembles the REST API described in docs/API-CONTRACT.md: routing,
// auth/RBAC wiring, and the resource handlers themselves.
package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"

	appmw "insurance-module/internal/api/middleware"
	"insurance-module/internal/audit"
	"insurance-module/internal/config"
	"insurance-module/internal/models"
	"insurance-module/internal/reports"
	"insurance-module/internal/ruleengine"
	"insurance-module/internal/workflow"
)

type Deps struct {
	DB       *gorm.DB
	Cfg      config.Config
	Rules    *ruleengine.Engine
	Workflow *workflow.Engine
	Audit    *audit.Service
	Reports  *reports.Service
}

func NewRouter(d Deps) http.Handler {
	h := &handlers{d: d}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   d.Cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", h.healthz)

	authed := appmw.Authenticate(d.Cfg.JWTSecret)
	adminOnly := appmw.RequireRole(models.RoleAdmin)
	adminOrReviewer := appmw.RequireRole(models.RoleAdmin, models.RoleReviewer)
	reviewOrAdmin := appmw.RequireRole(models.RoleReviewer, models.RoleAdmin)
	adminOrAuditor := appmw.RequireRole(models.RoleAdmin, models.RoleAuditor)
	anyRole := appmw.RequireRole(models.RoleAdmin, models.RoleReviewer, models.RoleEmployee, models.RoleAuditor)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", h.login)

		r.Group(func(r chi.Router) {
			r.Use(authed, anyRole)

			r.Get("/auth/me", h.me)

			r.Get("/service-types", h.listServiceTypes)
			r.Get("/contracts", h.listContracts)
			r.Get("/plans", h.listPlans)
			r.Get("/coverage-rules", h.listCoverageRules)

			r.Get("/employees/{id}", h.getEmployee)
			r.Get("/employees/{id}/dependents", h.listDependents)
			r.Get("/employees/{id}/remaining-caps", h.remainingCaps)

			r.Post("/claims", h.createClaim)
			r.Get("/claims", h.listClaims)
			r.Get("/claims/{id}", h.getClaim)
			r.Get("/claims/{id}/history", h.claimHistory)
			r.Post("/claims/{id}/submit", h.claimSubmit)
			r.Post("/claims/{id}/resubmit", h.claimResubmit)
		})

		r.Group(func(r chi.Router) {
			r.Use(authed, reviewOrAdmin)
			r.Post("/claims/{id}/start-review", h.claimStartReview)
			r.Post("/claims/{id}/approve", h.claimApprove)
			r.Post("/claims/{id}/reject", h.claimReject)
			r.Post("/claims/{id}/return-for-docs", h.claimReturnForDocs)
			r.Post("/claims/{id}/mark-paid", h.claimMarkPaid)
			r.Post("/claims/{id}/close", h.claimClose)
		})

		r.Group(func(r chi.Router) {
			r.Use(authed, adminOnly)
			r.Get("/employees", h.listEmployees)
			r.Post("/employees", h.createEmployee)
			r.Patch("/employees/{id}", h.updateEmployee)
			r.Post("/employees/{id}/dependents", h.createDependent)

			r.Post("/contracts", h.createContract)
			r.Post("/plans", h.createPlan)
			r.Post("/coverage-rules", h.createCoverageRule)

			r.Get("/admin/users", h.listUsers)
			r.Post("/admin/users", h.createUser)
			r.Patch("/admin/users/{id}", h.updateUser)
		})

		r.Group(func(r chi.Router) {
			r.Use(authed, adminOrAuditor)
			r.Get("/audit-logs", h.listAuditLogs)
			r.Get("/reports/summary", h.reportSummary)
			r.Get("/reports/spend-by-employee", h.reportSpendByEmployee)
			r.Get("/reports/spend-by-service-type", h.reportSpendByServiceType)
			r.Get("/reports/spend-by-month", h.reportSpendByMonth)
		})

		// reviewer/admin can also read employee list/detail implicitly via adminOnly group above
		// for mutation, but GET employees/{id} is already open to anyRole with an
		// ownership check inside the handler; adminOrReviewer kept for symmetry/future routes.
		_ = adminOrReviewer

		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAPIKey(d.DB))
			r.Post("/integration/employees/sync", h.integrationSyncEmployees)
			r.Get("/integration/claims/{id}/status", h.integrationClaimStatus)
		})
	})

	return r
}

func (h *handlers) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
