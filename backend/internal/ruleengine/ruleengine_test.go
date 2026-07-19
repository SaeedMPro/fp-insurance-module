package ruleengine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"insurance-module/internal/models"
)

func f(v float64) *float64 { return &v }

// TestCompute_FiveServiceTypes exercises Compute against the five seeded service
// types (outpatient visit, pharmacy, dental, hospitalization, optometry) under the
// Standard plan's rule parameters, matching the acceptance criterion that requires
// correct payable-amount/remaining-cap math for at least five distinct service
// types with different percentages and caps.
func TestCompute_FiveServiceTypes(t *testing.T) {
	cases := []struct {
		name            string
		rule            models.CoverageRule
		requested       float64
		usedAnnual      float64
		wantPayable     float64
		wantCappedClaim bool
		wantCappedYear  bool
	}{
		{
			name:        "outpatient visit under per-claim cap",
			rule:        models.CoverageRule{CoveragePercent: 70, PerClaimCap: f(500000), AnnualCap: f(5000000)},
			requested:   400000,
			usedAnnual:  0,
			wantPayable: 280000, // 70% of 400000
		},
		{
			name:            "pharmacy exceeds per-claim cap",
			rule:            models.CoverageRule{CoveragePercent: 80, PerClaimCap: f(1000000), AnnualCap: f(10000000)},
			requested:       2000000,
			usedAnnual:      0,
			wantPayable:     1000000, // 80% of 2,000,000 = 1,600,000 > cap 1,000,000
			wantCappedClaim: true,
		},
		{
			name:        "dental partial coverage",
			rule:        models.CoverageRule{CoveragePercent: 50, PerClaimCap: f(3000000), AnnualCap: f(15000000)},
			requested:   1000000,
			usedAnnual:  0,
			wantPayable: 500000,
		},
		{
			name:            "hospitalization hits both per-claim and remaining annual cap",
			rule:            models.CoverageRule{CoveragePercent: 90, PerClaimCap: f(50000000), AnnualCap: f(100000000)},
			requested:       60000000, // 90% = 54,000,000 > per-claim cap 50,000,000
			usedAnnual:      98000000, // only 2,000,000 of annual cap remaining
			wantPayable:     2000000,
			wantCappedClaim: true,
			wantCappedYear:  true,
		},
		{
			name:        "optometry within all caps",
			rule:        models.CoverageRule{CoveragePercent: 60, PerClaimCap: f(2000000), AnnualCap: f(4000000)},
			requested:   1500000,
			usedAnnual:  1000000,
			wantPayable: 900000, // 60% of 1,500,000 = 900,000, remaining cap 3,000,000 — not capped
		},
		{
			name:           "annual cap already exhausted pays zero",
			rule:           models.CoverageRule{CoveragePercent: 70, PerClaimCap: f(500000), AnnualCap: f(5000000)},
			requested:      400000,
			usedAnnual:     5000000,
			wantPayable:    0,
			wantCappedYear: true,
		},
		{
			name:        "no caps configured (unlimited plan)",
			rule:        models.CoverageRule{CoveragePercent: 100, PerClaimCap: nil, AnnualCap: nil},
			requested:   123456.78,
			usedAnnual:  0,
			wantPayable: 123456.78,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Compute(tc.rule, tc.requested, tc.usedAnnual)
			assert.InDelta(t, tc.wantPayable, got.PayableAmount, 0.01)
			assert.Equal(t, tc.wantCappedClaim, got.CappedByPerClaim)
			assert.Equal(t, tc.wantCappedYear, got.CappedByAnnualCap)
			assert.GreaterOrEqual(t, got.PayableAmount, 0.0)
		})
	}
}

func TestCompute_RemainingCapAfterClaim(t *testing.T) {
	rule := models.CoverageRule{CoveragePercent: 80, PerClaimCap: f(1000000), AnnualCap: f(10000000)}
	got := Compute(rule, 1000000, 3000000) // covered = 800000, remaining before = 7,000,000
	assert.InDelta(t, 800000, got.PayableAmount, 0.01)
	if assert.NotNil(t, got.RemainingAnnualCapAfter) {
		assert.InDelta(t, 6200000, *got.RemainingAnnualCapAfter, 0.01)
	}
}

func TestContainsRelation(t *testing.T) {
	assert.True(t, containsRelation([]string{"self", "spouse"}, models.RelationSpouse))
	assert.False(t, containsRelation([]string{"self"}, models.RelationChild))
}
