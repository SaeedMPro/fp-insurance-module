package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/require"

	"insurance-module/internal/app"
	"insurance-module/internal/app/apptest"
	"insurance-module/internal/domain"
	"insurance-module/internal/service/claims"
	"insurance-module/internal/service/employees"
	"insurance-module/internal/service/users"
	"insurance-module/internal/storage/postgres"
	transporthttp "insurance-module/internal/transport/http"
)

// TestOpenAPIConformance boots the real router over a real (rolled-back)
// database and validates actual responses against backend/api/openapi.yaml.
// The spec is the contract (ADR-0002); this test is what keeps the running
// server honest about it, so a handler cannot silently drift from the schema
// the frontend's types are generated from.
func TestOpenAPIConformance(t *testing.T) {
	store, _ := apptest.Open(t)
	ctx := context.Background()

	const secret = "conformance-test-secret"
	svcs := app.Build(store, app.Options{JWTSecret: secret, JWTTTL: time.Hour})

	const (
		adminUser  = "oa-admin"
		adminPass  = "Admin123!"
		empUser    = "oa-employee"
		empPass    = "Employee123!"
	)
	employeeID, claimID := setupConformanceData(t, ctx, store, svcs, adminUser, adminPass, empUser, empPass)

	handler := transporthttp.NewRouter(transporthttp.Config{
		JWTSecret:   secret,
		CORSOrigins: []string{"*"},
	}, svcs)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	loader := &openapi3.Loader{Context: ctx, IsExternalRefsAllowed: true}
	doc, err := loader.LoadFromFile("../../../api/openapi.yaml")
	require.NoError(t, err, "openapi.yaml must parse")
	require.NoError(t, doc.Validate(ctx), "openapi.yaml must be a valid OpenAPI document")

	doc.Servers = openapi3.Servers{{URL: srv.URL + "/api/v1"}}
	specRouter, err := gorillamux.NewRouter(doc)
	require.NoError(t, err)

	client := srv.Client()

	adminToken := login(t, client, srv.URL, adminUser, adminPass)
	employeeToken := login(t, client, srv.URL, empUser, empPass)

	cases := []struct {
		name   string
		method string
		path   string
		token  string
		body   any
		status int
	}{
		{"login", "POST", "/api/v1/auth/login", "", map[string]string{"username": adminUser, "password": adminPass}, 200},
		{"me", "GET", "/api/v1/auth/me", adminToken, nil, 200},
		{"service types", "GET", "/api/v1/service-types", adminToken, nil, 200},
		{"contracts", "GET", "/api/v1/contracts", adminToken, nil, 200},
		{"plans", "GET", "/api/v1/plans", adminToken, nil, 200},
		{"coverage rules", "GET", "/api/v1/coverage-rules", adminToken, nil, 200},
		{"employees", "GET", "/api/v1/employees?page_size=5", adminToken, nil, 200},
		{"employee", "GET", "/api/v1/employees/" + employeeID, adminToken, nil, 200},
		{"dependents", "GET", "/api/v1/employees/" + employeeID + "/dependents", adminToken, nil, 200},
		{"remaining caps", "GET", "/api/v1/employees/" + employeeID + "/remaining-caps", adminToken, nil, 200},
		{"claims", "GET", "/api/v1/claims?page_size=5", adminToken, nil, 200},
		{"claim", "GET", "/api/v1/claims/" + claimID, adminToken, nil, 200},
		{"claim history", "GET", "/api/v1/claims/" + claimID + "/history", adminToken, nil, 200},
		{"audit logs", "GET", "/api/v1/audit-logs?page_size=5", adminToken, nil, 200},
		{"report summary", "GET", "/api/v1/reports/summary", adminToken, nil, 200},
		{"spend by employee", "GET", "/api/v1/reports/spend-by-employee", adminToken, nil, 200},
		{"spend by service type", "GET", "/api/v1/reports/spend-by-service-type", adminToken, nil, 200},
		{"spend by month", "GET", "/api/v1/reports/spend-by-month", adminToken, nil, 200},
		{"users", "GET", "/api/v1/admin/users", adminToken, nil, 200},
		{"bad credentials", "POST", "/api/v1/auth/login", "", map[string]string{"username": adminUser, "password": "wrong"}, 401},
		{"claim not found", "GET", "/api/v1/claims/00000000-0000-0000-0000-000000000000", adminToken, nil, 404},
		{"role denied", "POST", "/api/v1/coverage-rules", employeeToken, map[string]any{}, 403},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := newRequest(t, srv.URL, tc.method, tc.path, tc.token, tc.body)

			route, pathParams, err := specRouter.FindRoute(req)
			require.NoError(t, err, "%s %s must be described by the spec", tc.method, tc.path)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			payload := readBody(t, resp)
			require.Equal(t, tc.status, resp.StatusCode, "unexpected status; body=%s", payload)

			err = openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{
					Request:    req,
					PathParams: pathParams,
					Route:      route,
					Options:    &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
				},
				Status:  resp.StatusCode,
				Header:  resp.Header,
				Body:    io.NopCloser(bytes.NewReader(payload)),
				Options: &openapi3filter.Options{IncludeResponseStatus: true},
			})
			require.NoError(t, err, "response does not conform to the spec; body=%s", payload)
		})
	}
}

// setupConformanceData creates the minimal admin, employee, and claim needed
// for the OpenAPI walk — test-only, not a shared seed package.
func setupConformanceData(
	t *testing.T,
	ctx context.Context,
	store *postgres.Store,
	svcs transporthttp.Services,
	adminUser, adminPass, empUser, empPass string,
) (employeeID, claimID string) {
	t.Helper()
	suffix := fmt.Sprintf("%012d", time.Now().UnixNano()%1e12)

	_, err := svcs.Users.Create(ctx, users.CreateInput{
		Username: adminUser, Password: adminPass,
		FullName: "OA Admin", Role: domain.RoleAdmin,
	})
	require.NoError(t, err)

	plan, err := store.GetPlanByName(ctx, "استاندارد")
	require.NoError(t, err, "db/seed.sql must be applied before integration tests")

	emp, err := svcs.Employees.Create(ctx, employees.CreateInput{
		PersonnelNo: "OA-" + suffix,
		FullName:    "OA Employee",
		NationalID:  "OA-NID-" + suffix,
		HireDate:    time.Now().AddDate(-2, 0, 0),
		PlanID:      &plan.ID,
	})
	require.NoError(t, err)

	empAccount, err := svcs.Users.Create(ctx, users.CreateInput{
		Username: empUser, Password: empPass,
		FullName: emp.FullName, Role: domain.RoleEmployee, EmployeeID: &emp.ID,
	})
	require.NoError(t, err)

	st, err := store.GetServiceTypeByCode(ctx, "outpatient_visit")
	require.NoError(t, err)

	actor := domain.Actor{UserID: empAccount.ID, Username: empAccount.Username, Role: empAccount.Role}
	claim, err := svcs.Claims.Create(ctx, actor, claims.CreateInput{
		BeneficiaryType: domain.BeneficiarySelf,
		ServiceTypeID:   st.ID,
		RequestedAmount: 350000,
		ReceiptDate:     time.Now().AddDate(0, 0, -1),
		Description:     "openapi conformance claim",
	})
	require.NoError(t, err)
	_, err = svcs.Claims.Submit(ctx, actor, claim.ID)
	require.NoError(t, err)

	return emp.ID.String(), claim.ID.String()
}

// TestOpenAPISpecCoversEveryRoute guards the other direction: every route the
// router serves must be documented, so the spec can never fall behind the code.
func TestOpenAPISpecCoversEveryRoute(t *testing.T) {
	loader := &openapi3.Loader{Context: context.Background()}
	doc, err := loader.LoadFromFile("../../../api/openapi.yaml")
	require.NoError(t, err)

	documented := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			documented[method+" /api/v1"+path] = true
		}
	}

	for _, r := range transporthttp.Routes() {
		if r == "GET /healthz" { // infrastructure endpoint, deliberately outside the versioned API
			continue
		}
		require.True(t, documented[r], "route %q is served but missing from api/openapi.yaml", r)
	}
}

func login(t *testing.T, client *http.Client, base, username, password string) string {
	t.Helper()
	req := newRequest(t, base, "POST", "/api/v1/auth/login", "",
		map[string]string{"username": username, "password": password})
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, 200, resp.StatusCode)
	var out struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(readBody(t, resp), &out))
	require.NotEmpty(t, out.Token)
	return out.Token
}

func newRequest(t *testing.T, base, method, path, token string, body any) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, base+path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	require.NoError(t, err)
	return buf.Bytes()
}
