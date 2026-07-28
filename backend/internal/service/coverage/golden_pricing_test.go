package coverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"insurance-module/internal/domain"
)

// Golden pricing tests. These exist to freeze the engine's arithmetic across
// refactors — in particular the float64 → integer-Rial migration (ADR-0003).
// The table below is deliberately exhaustive about the interesting boundaries
// (fractional percentages, .5 rounding, cap collisions, exhausted caps); the
// expected values were captured from the pre-migration implementation, so any
// behaviour change shows up as a diff instead of silently re-pricing claims.
//
// Regenerate deliberately (only when a change is intended and reviewed):
//	UPDATE_GOLDEN=1 go test ./internal/service/coverage/ -run TestGoldenPricing

type goldenCase struct {
	Name             string   `json:"name"`
	CoveragePercent  float64  `json:"coverage_percent"`
	PerClaimCap      *float64 `json:"per_claim_cap"`
	AnnualCap        *float64 `json:"annual_cap"`
	RequestedAmount  float64  `json:"requested_amount"`
	UsedAnnual       float64  `json:"used_annual"`
	WantPayable      float64  `json:"want_payable"`
	WantCappedClaim  bool     `json:"want_capped_per_claim"`
	WantCappedAnnual bool     `json:"want_capped_annual"`
	WantRemainingCap *float64 `json:"want_remaining_annual_after"`
}

func goldenCases() []goldenCase {
	// Inputs only — expectations are filled in from the golden file (or
	// recomputed when UPDATE_GOLDEN=1).
	return []goldenCase{
		{Name: "seeded outpatient 70pct under caps", CoveragePercent: 70, PerClaimCap: p(500000), AnnualCap: p(5000000), RequestedAmount: 400000},
		{Name: "seeded pharmacy 80pct over per-claim cap", CoveragePercent: 80, PerClaimCap: p(1000000), AnnualCap: p(10000000), RequestedAmount: 2000000},
		{Name: "seeded dental 50pct", CoveragePercent: 50, PerClaimCap: p(3000000), AnnualCap: p(15000000), RequestedAmount: 1000000},
		{Name: "seeded hospitalization both caps bind", CoveragePercent: 90, PerClaimCap: p(50000000), AnnualCap: p(100000000), RequestedAmount: 60000000, UsedAnnual: 98000000},
		{Name: "seeded optometry 60pct", CoveragePercent: 60, PerClaimCap: p(2000000), AnnualCap: p(4000000), RequestedAmount: 1500000, UsedAnnual: 1000000},
		{Name: "premium hospitalization 95pct", CoveragePercent: 95, PerClaimCap: p(80000000), AnnualCap: p(150000000), RequestedAmount: 12345678},

		// Rounding boundaries: odd amounts and fractional percents.
		{Name: "odd amount 33pct", CoveragePercent: 33, RequestedAmount: 1001},
		{Name: "third-ish 33.33pct", CoveragePercent: 33.33, RequestedAmount: 100000},
		{Name: "half rial exact .5 rounds up", CoveragePercent: 50, RequestedAmount: 1001},
		{Name: "fractional percent 12.5", CoveragePercent: 12.5, RequestedAmount: 8001},
		{Name: "one rial at 50pct", CoveragePercent: 50, RequestedAmount: 1},
		{Name: "three rial at 33.33pct", CoveragePercent: 33.33, RequestedAmount: 3},

		// Cap interactions.
		{Name: "annual cap exhausted pays zero", CoveragePercent: 70, PerClaimCap: p(500000), AnnualCap: p(5000000), RequestedAmount: 400000, UsedAnnual: 5000000},
		{Name: "annual overspent clamps to zero", CoveragePercent: 70, AnnualCap: p(1000000), RequestedAmount: 400000, UsedAnnual: 1500000},
		{Name: "annual cap leaves odd remainder", CoveragePercent: 80, PerClaimCap: p(9000000), AnnualCap: p(1000001), RequestedAmount: 5000000, UsedAnnual: 500000},
		{Name: "per-claim cap exactly equals covered", CoveragePercent: 50, PerClaimCap: p(500000), RequestedAmount: 1000000},
		{Name: "zero percent coverage", CoveragePercent: 0, RequestedAmount: 1000000},
		{Name: "full coverage no caps", CoveragePercent: 100, RequestedAmount: 123456},
		{Name: "large hospitalization no caps", CoveragePercent: 90, RequestedAmount: 987654321},
	}
}

func p(v float64) *float64 { return &v }

func TestGoldenPricing(t *testing.T) {
	path := filepath.Join("testdata", "golden_pricing.json")
	cases := goldenCases()

	// Compute what the engine produces now.
	got := make([]goldenCase, 0, len(cases))
	for _, tc := range cases {
		rule := domain.CoverageRule{
			CoveragePercent: tc.CoveragePercent,
			PerClaimCap:     tc.PerClaimCap,
			AnnualCap:       tc.AnnualCap,
		}
		res := Compute(rule, tc.RequestedAmount, tc.UsedAnnual)
		out := tc
		out.WantPayable = res.PayableAmount
		out.WantCappedClaim = res.CappedByPerClaim
		out.WantCappedAnnual = res.CappedByAnnualCap
		out.WantRemainingCap = res.RemainingAnnualCapAfter
		got = append(got, out)
	}

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(t, os.MkdirAll("testdata", 0o750))
		blob, err := json.MarshalIndent(got, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, append(blob, '\n'), 0o600))
		t.Logf("golden file updated: %s (%d cases)", path, len(got))
		return
	}

	raw, err := os.ReadFile(path) // #nosec G304 -- fixed test-data path
	require.NoError(t, err, "golden file missing; run with UPDATE_GOLDEN=1 to create it")
	var want []goldenCase
	require.NoError(t, json.Unmarshal(raw, &want))
	require.Len(t, got, len(want), "case count changed; regenerate the golden file deliberately")

	for i := range want {
		t.Run(want[i].Name, func(t *testing.T) {
			require.Equal(t, want[i].Name, got[i].Name, "case order must be stable")
			require.Equal(t, want[i].WantPayable, got[i].WantPayable, "payable amount changed")
			require.Equal(t, want[i].WantCappedClaim, got[i].WantCappedClaim, "per-claim cap flag changed")
			require.Equal(t, want[i].WantCappedAnnual, got[i].WantCappedAnnual, "annual cap flag changed")
			require.Equal(t, want[i].WantRemainingCap, got[i].WantRemainingCap, "remaining annual cap changed")
		})
	}
}
