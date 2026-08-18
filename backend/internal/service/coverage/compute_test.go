package coverage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"insurance-module/internal/domain"
)

// f builds an optional cap; caps are whole rial.
func f(v domain.Rial) *domain.Rial { return &v }

// TestCompute_FiveServiceTypes exercises Compute against the five seeded
// service types under the Standard plan's parameters, matching the acceptance
// criterion requiring correct payable/remaining-cap math for at least five
// service types with different percentages and caps.
func TestCompute_FiveServiceTypes(t *testing.T) {
	cases := []struct {
		name            string
		rule            domain.CoverageRule
		requested       domain.Rial
		usedAnnual      domain.Rial
		wantPayable     domain.Rial
		wantCappedClaim bool
		wantCappedYear  bool
	}{
		{
			name:        "outpatient visit under per-claim cap",
			rule:        domain.CoverageRule{CoveragePercent: domain.PercentFromFloat(70), PerClaimCap: f(500000), AnnualCap: f(5000000)},
			requested:   400000,
			usedAnnual:  0,
			wantPayable: 280000, // 70% of 400,000
		},
		{
			name:            "pharmacy exceeds per-claim cap",
			rule:            domain.CoverageRule{CoveragePercent: domain.PercentFromFloat(80), PerClaimCap: f(1000000), AnnualCap: f(10000000)},
			requested:       2000000,
			usedAnnual:      0,
			wantPayable:     1000000, // 80% = 1,600,000 > cap 1,000,000
			wantCappedClaim: true,
		},
		{
			name:        "dental partial coverage",
			rule:        domain.CoverageRule{CoveragePercent: domain.PercentFromFloat(50), PerClaimCap: f(3000000), AnnualCap: f(15000000)},
			requested:   1000000,
			usedAnnual:  0,
			wantPayable: 500000,
		},
		{
			name:            "hospitalization hits both per-claim and remaining annual cap",
			rule:            domain.CoverageRule{CoveragePercent: domain.PercentFromFloat(90), PerClaimCap: f(50000000), AnnualCap: f(100000000)},
			requested:       60000000, // 90% = 54,000,000 > per-claim cap 50,000,000
			usedAnnual:      98000000, // only 2,000,000 of annual cap remaining
			wantPayable:     2000000,
			wantCappedClaim: true,
			wantCappedYear:  true,
		},
		{
			name:        "optometry within all caps",
			rule:        domain.CoverageRule{CoveragePercent: domain.PercentFromFloat(60), PerClaimCap: f(2000000), AnnualCap: f(4000000)},
			requested:   1500000,
			usedAnnual:  1000000,
			wantPayable: 900000,
		},
		{
			name:           "annual cap already exhausted pays zero",
			rule:           domain.CoverageRule{CoveragePercent: domain.PercentFromFloat(70), PerClaimCap: f(500000), AnnualCap: f(5000000)},
			requested:      400000,
			usedAnnual:     5000000,
			wantPayable:    0,
			wantCappedYear: true,
		},
		{
			name:        "no caps configured (unlimited plan)",
			rule:        domain.CoverageRule{CoveragePercent: domain.PercentFromFloat(100), PerClaimCap: nil, AnnualCap: nil},
			requested:   123457,
			usedAnnual:  0,
			wantPayable: 123457,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Compute(tc.rule, tc.requested, tc.usedAnnual)
			assert.Equal(t, tc.wantPayable, got.PayableAmount)
			assert.Equal(t, tc.wantCappedClaim, got.CappedByPerClaim)
			assert.Equal(t, tc.wantCappedYear, got.CappedByAnnualCap)
			assert.GreaterOrEqual(t, got.PayableAmount, domain.Rial(0))
		})
	}
}

func TestCompute_RemainingCapAfterClaim(t *testing.T) {
	rule := domain.CoverageRule{CoveragePercent: domain.PercentFromFloat(80), PerClaimCap: f(1000000), AnnualCap: f(10000000)}
	got := Compute(rule, 1000000, 3000000) // covered 800,000; remaining before 7,000,000
	assert.Equal(t, domain.Rial(800000), got.PayableAmount)
	if assert.NotNil(t, got.RemainingAnnualCapAfter) {
		assert.Equal(t, domain.Rial(6200000), *got.RemainingAnnualCapAfter)
	}
}

func TestRule_EligibleFor(t *testing.T) {
	rule := domain.CoverageRule{EligibleRelations: []domain.Relation{domain.RelationSelf, domain.RelationSpouse}}
	assert.True(t, rule.EligibleFor(domain.RelationSpouse))
	assert.False(t, rule.EligibleFor(domain.RelationChild))
}
