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

func TestUpdateResetsPassword(t *testing.T) {
	_, svcs := apptest.Open(t)
	u, err := svcs.Users.Create(context.Background(), users.CreateInput{
		Username: "pw_reset_me",
		Password: "Reviewer1!",
		FullName: "Reset Target",
		Role:     domain.RoleReviewer,
	})
	require.NoError(t, err)

	next := "NewPass99!"
	_, err = svcs.Users.Update(context.Background(), u.ID, users.UpdateInput{Password: &next})
	require.NoError(t, err)

	_, _, err = svcs.Users.Login(context.Background(), "pw_reset_me", "Reviewer1!")
	require.ErrorIs(t, err, users.ErrBadCredentials)

	_, token, err := svcs.Users.Login(context.Background(), "pw_reset_me", next)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestUpdateRejectsShortPassword(t *testing.T) {
	_, svcs := apptest.Open(t)
	u, err := svcs.Users.Create(context.Background(), users.CreateInput{
		Username: "short_pw",
		Password: "Reviewer1!",
		FullName: "Short",
		Role:     domain.RoleReviewer,
	})
	require.NoError(t, err)

	short := "abc"
	_, err = svcs.Users.Update(context.Background(), u.ID, users.UpdateInput{Password: &short})
	require.Error(t, err)
	require.Contains(t, err.Error(), "password must be at least")
}

func TestUpdateRejectsDeactivateAdmin(t *testing.T) {
	_, svcs := apptest.Open(t)
	admin, created, err := svcs.Users.EnsureAdmin(context.Background(), users.CreateInput{
		Username: "admin",
		Password: "Admin123!",
		FullName: "مدیر سامانه",
	})
	require.NoError(t, err)
	require.False(t, created)

	inactive := false
	_, err = svcs.Users.Update(context.Background(), admin.ID, users.UpdateInput{IsActive: &inactive})
	require.ErrorIs(t, err, users.ErrAdminDeactivateForbidden)
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
