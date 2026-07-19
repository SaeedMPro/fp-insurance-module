// Package users owns authentication (login → JWT) and user administration.
package users

import (
	"context"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/auth"
	"insurance-module/internal/domain"
)

var ErrBadCredentials = domain.Unauthorizedf("invalid username or password")

type Repo interface {
	GetUser(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (domain.User, error)
	ListUsers(ctx context.Context) ([]domain.User, error)
	CreateUser(ctx context.Context, u *domain.User) error
	SaveUser(ctx context.Context, u *domain.User) error
	InsertAudit(ctx context.Context, entry *domain.AuditLog) error
}

type Service struct {
	repo      Repo
	jwtSecret string
	jwtTTL    time.Duration
	clock     domain.Clock
}

func NewService(repo Repo, jwtSecret string, jwtTTL time.Duration, clock domain.Clock) *Service {
	if clock == nil {
		clock = domain.SystemClock{}
	}
	return &Service{repo: repo, jwtSecret: jwtSecret, jwtTTL: jwtTTL, clock: clock}
}

// Login verifies credentials, records the login in the audit trail, and
// returns the user plus a signed JWT.
func (s *Service) Login(ctx context.Context, username, password string) (domain.User, string, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		// Deliberately indistinguishable from a wrong password.
		return domain.User{}, "", ErrBadCredentials
	}
	if !user.IsActive || !auth.CheckPassword(user.PasswordHash, password) {
		return domain.User{}, "", ErrBadCredentials
	}

	token, err := auth.GenerateToken(s.jwtSecret, s.jwtTTL, auth.TokenSubject{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
	if err != nil {
		return domain.User{}, "", domain.Internalf(err, "could not issue token")
	}

	// Login is auditable but must not fail the login itself.
	_ = s.repo.InsertAudit(ctx, &domain.AuditLog{
		EntityType:    "user",
		EntityID:      user.ID.String(),
		Action:        "login",
		ActorUserID:   &user.ID,
		ActorUsername: user.Username,
		OccurredAt:    s.clock.Now(),
	})
	return user, token, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return s.repo.GetUser(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.ListUsers(ctx)
}

type CreateInput struct {
	Username   string
	Password   string
	FullName   string
	Role       domain.Role
	EmployeeID *uuid.UUID
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.User, error) {
	if in.Username == "" || in.Password == "" || in.FullName == "" {
		return domain.User{}, domain.Validationf("username, password and full name are required")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return domain.User{}, domain.Internalf(err, "could not hash password")
	}
	u := domain.User{
		Username:     in.Username,
		PasswordHash: hash,
		FullName:     in.FullName,
		Role:         in.Role,
		EmployeeID:   in.EmployeeID,
		IsActive:     true,
	}
	if err := s.repo.CreateUser(ctx, &u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

type UpdateInput struct {
	Role     *domain.Role
	IsActive *bool
	Password *string
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (domain.User, error) {
	u, err := s.repo.GetUser(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if in.Role != nil {
		u.Role = *in.Role
	}
	if in.IsActive != nil {
		u.IsActive = *in.IsActive
	}
	if in.Password != nil && *in.Password != "" {
		hash, err := auth.HashPassword(*in.Password)
		if err != nil {
			return domain.User{}, domain.Internalf(err, "could not hash password")
		}
		u.PasswordHash = hash
	}
	if err := s.repo.SaveUser(ctx, &u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}
