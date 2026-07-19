package postgres

import (
	"context"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

func (s *Store) GetUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	var row userRow
	if err := s.ctx(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.User{}, notFound(err, "user")
	}
	return row.toDomain(), nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (domain.User, error) {
	var row userRow
	if err := s.ctx(ctx).Where("username = ?", username).First(&row).Error; err != nil {
		return domain.User{}, notFound(err, "user")
	}
	return row.toDomain(), nil
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	var rows []userRow
	if err := s.ctx(ctx).Order("username").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (s *Store) CreateUser(ctx context.Context, u *domain.User) error {
	row := userFromDomain(*u)
	if err := s.ctx(ctx).Create(&row).Error; err != nil {
		return err
	}
	*u = row.toDomain()
	return nil
}

func (s *Store) SaveUser(ctx context.Context, u *domain.User) error {
	row := userFromDomain(*u)
	if err := s.ctx(ctx).Save(&row).Error; err != nil {
		return err
	}
	*u = row.toDomain()
	return nil
}
