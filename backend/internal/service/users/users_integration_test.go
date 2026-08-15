package users_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"insurance-module/internal/app/apptest"
	"insurance-module/internal/domain"
	"insurance-module/internal/service/users"
)

func TestCreateRejectsAdminRole(t *testing.T) {
	_, svcs := apptest.Open(t)
	_, err := svcs.Users.Create(context.Background(), users.CreateInput{
		Username: "would_be_admin",
		Password: "Admin123!",
		FullName: "Should Fail",
		Role:     domain.RoleAdmin,
	})
	require.ErrorIs(t, err, users.ErrAdminCreateForbidden)
}

func TestUpdateRejectsPromoteToAdmin(t *testing.T) {
	_, svcs := apptest.Open(t)
	u, err := svcs.Users.Create(context.Background(), users.CreateInput{
		Username: "promote_me",
		Password: "Reviewer1!",
		FullName: "Reviewer",
		Role:     domain.RoleReviewer,
	})
	require.NoError(t, err)

	role := domain.RoleAdmin
	_, err = svcs.Users.Update(context.Background(), u.ID, users.UpdateInput{Role: &role})
	require.ErrorIs(t, err, users.ErrAdminPromoteForbidden)
}

func TestUpdateRejectsDemoteAdmin(t *testing.T) {
	_, svcs := apptest.Open(t)
	admin, created, err := svcs.Users.EnsureAdmin(context.Background(), users.CreateInput{
		Username: "admin",
		Password: "Admin123!",
		FullName: "مدیر سامانه",
	})
	require.NoError(t, err)
	require.False(t, created)

	role := domain.RoleReviewer
	_, err = svcs.Users.Update(context.Background(), admin.ID, users.UpdateInput{Role: &role})
	require.ErrorIs(t, err, users.ErrAdminDemoteForbidden)
}

func TestCreateEmployeeRequiresLink(t *testing.T) {
	_, svcs := apptest.Open(t)
	_, err := svcs.Users.Create(context.Background(), users.CreateInput{
		Username: "orphan_employee",
		Password: "Employee123!",
		FullName: "بدون پرونده",
		Role:     domain.RoleEmployee,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "employee role requires a linked employee record")
}

func TestUpdateRejectsEmployeeRoleWithoutLink(t *testing.T) {
	_, svcs := apptest.Open(t)
	u, err := svcs.Users.Create(context.Background(), users.CreateInput{
		Username: "staff_then_emp",
		Password: "Reviewer1!",
		FullName: "کارشناس",
		Role:     domain.RoleReviewer,
	})
	require.NoError(t, err)

	role := domain.RoleEmployee
	_, err = svcs.Users.Update(context.Background(), u.ID, users.UpdateInput{Role: &role})
	require.Error(t, err)
	require.Contains(t, err.Error(), "employee role requires a linked employee record")
}

func TestEnsureAdminIdempotent(t *testing.T) {
	_, svcs := apptest.Open(t)
	in := users.CreateInput{
		Username: "admin",
		Password: "Admin123!",
		FullName: "مدیر سامانه",
	}
	first, created1, err := svcs.Users.EnsureAdmin(context.Background(), in)
	require.NoError(t, err)
	require.False(t, created1)
	require.Equal(t, domain.RoleAdmin, first.Role)

	second, created2, err := svcs.Users.EnsureAdmin(context.Background(), in)
	require.NoError(t, err)
	require.False(t, created2)
	require.Equal(t, first.ID, second.ID)
}
