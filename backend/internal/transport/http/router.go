package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"insurance-module/internal/domain"
	"insurance-module/internal/platform/logging"
	"insurance-module/internal/service/audit"
	"insurance-module/internal/service/claims"
	"insurance-module/internal/service/coverage"
	"insurance-module/internal/service/employees"
	"insurance-module/internal/service/integration"
	"insurance-module/internal/service/reports"
	"insurance-module/internal/service/users"
)

// Server bundles the services the HTTP layer exposes.
type Server struct {
	users       *users.Service
	claims      *claims.Service
	coverage    *coverage.Service
	employees   *employees.Service
	audit       *audit.Service
	reports     *reports.Service
	integration *integration.Service
}

// Config is what the router needs beyond the services themselves.
type Config struct {
	JWTSecret   string
	CORSOrigins []string
	Logger      *slog.Logger
}

type Services struct {
	Users       *users.Service
	Claims      *claims.Service
	Coverage    *coverage.Service
	Employees   *employees.Service
	Audit       *audit.Service
	Reports     *reports.Service
	Integration *integration.Service
}

// NewRouter assembles the REST API described in docs/API-CONTRACT.md.
func NewRouter(cfg Config, svcs Services) http.Handler {
	s := &Server{
		users:       svcs.Users,
		claims:      svcs.Claims,
		coverage:    svcs.Coverage,
		employees:   svcs.Employees,
		audit:       svcs.Audit,
		reports:     svcs.Reports,
		integration: svcs.Integration,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	if cfg.Logger != nil {
		r.Use(logging.RequestLogger(cfg.Logger))
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	authed := authenticate(cfg.JWTSecret)
	adminOnly := requireRole(domain.RoleAdmin)
	staff := requireRole(domain.RoleReviewer, domain.RoleAdmin)
	adminOrAuditor := requireRole(domain.RoleAdmin, domain.RoleAuditor)
	anyRole := requireRole(domain.RoleAdmin, domain.RoleReviewer, domain.RoleEmployee, domain.RoleAuditor)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.handleLogin)

		r.Group(func(r chi.Router) {
			r.Use(authed, anyRole)

			r.Get("/auth/me", s.handleMe)

			r.Get("/service-types", s.handleListServiceTypes)
			r.Get("/contracts", s.handleListContracts)
			r.Get("/plans", s.handleListPlans)
			r.Get("/coverage-rules", s.handleListCoverageRules)

			r.Get("/employees/{id}", s.handleGetEmployee)
			r.Get("/employees/{id}/dependents", s.handleListDependents)
			r.Get("/employees/{id}/remaining-caps", s.handleRemainingCaps)

			r.Post("/claims", s.handleCreateClaim)
			r.Get("/claims", s.handleListClaims)
			r.Get("/claims/{id}", s.handleGetClaim)
			r.Get("/claims/{id}/history", s.handleClaimHistory)
			r.Post("/claims/{id}/submit", s.transition(func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
				return s.claims.Submit(r.Context(), actor, id)
			}))
			r.Post("/claims/{id}/resubmit", s.transition(func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
				return s.claims.Resubmit(r.Context(), actor, id)
			}))
		})

		r.Group(func(r chi.Router) {
			r.Use(authed, staff)
			// The contract grants the employee LIST to reviewers as well as
			// admins (the old router mistakenly made it admin-only while the
			// reviewer UI links to it).
			r.Get("/employees", s.handleListEmployees)
			r.Post("/claims/{id}/start-review", s.transition(func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
				return s.claims.StartReview(r.Context(), actor, id)
			}))
			r.Post("/claims/{id}/approve", s.transition(func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
				return s.claims.Approve(r.Context(), actor, id)
			}))
			r.Post("/claims/{id}/reject", s.transition(func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
				reason, err := withReason(r)
				if err != nil {
					return domain.Claim{}, err
				}
				return s.claims.Reject(r.Context(), actor, id, reason)
			}))
			r.Post("/claims/{id}/return-for-docs", s.transition(func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
				reason, err := withReason(r)
				if err != nil {
					return domain.Claim{}, err
				}
				return s.claims.ReturnForDocs(r.Context(), actor, id, reason)
			}))
			r.Post("/claims/{id}/mark-paid", s.transition(func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
				return s.claims.MarkPaid(r.Context(), actor, id)
			}))
			r.Post("/claims/{id}/close", s.transition(func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
				return s.claims.Close(r.Context(), actor, id)
			}))
		})

		r.Group(func(r chi.Router) {
			r.Use(authed, adminOnly)
			r.Post("/employees", s.handleCreateEmployee)
			r.Patch("/employees/{id}", s.handleUpdateEmployee)
			r.Post("/employees/{id}/dependents", s.handleCreateDependent)

			r.Post("/contracts", s.handleCreateContract)
			r.Post("/plans", s.handleCreatePlan)
			r.Post("/coverage-rules", s.handleCreateCoverageRule)

			r.Get("/admin/users", s.handleListUsers)
			r.Post("/admin/users", s.handleCreateUser)
			r.Patch("/admin/users/{id}", s.handleUpdateUser)
		})

		r.Group(func(r chi.Router) {
			r.Use(authed, adminOrAuditor)
			r.Get("/audit-logs", s.handleListAuditLogs)
			r.Get("/reports/summary", s.handleReportSummary)
			r.Get("/reports/spend-by-employee", s.handleSpendByEmployee)
			r.Get("/reports/spend-by-service-type", s.handleSpendByServiceType)
			r.Get("/reports/spend-by-month", s.handleSpendByMonth)
		})

		r.Group(func(r chi.Router) {
			r.Use(requireAPIKey(svcs.Integration))
			r.Post("/integration/employees/sync", s.handleSyncEmployees)
			r.Get("/integration/claims/{id}/status", s.handleIntegrationClaimStatus)
		})
	})

	return r
}
