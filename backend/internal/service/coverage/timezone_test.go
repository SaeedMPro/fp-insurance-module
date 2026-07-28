package coverage

import (
	"testing"
	"time"

	"insurance-module/internal/domain"
)

// These tests pin the day-boundary behaviour that ADR-0004 exists for: business
// days are measured in Asia/Tehran, so a claim prices the same no matter which
// timezone the server runs in. Before the fix, contractYearWindow built its
// window in UTC and waiting periods compared raw instants, meaning a receipt
// near midnight could fall in a different contract year (or fail eligibility)
// depending purely on host configuration.

func TestContractYearWindow_IsStableAcrossHostTimezones(t *testing.T) {
	effectiveFrom := time.Date(2025, 3, 21, 0, 0, 0, 0, time.UTC)

	// The same instant, expressed in wildly different zones.
	instant := time.Date(2026, 7, 19, 21, 30, 0, 0, time.UTC)
	zones := []*time.Location{
		time.UTC,
		mustLoad(t, "Asia/Tehran"),
		mustLoad(t, "America/Los_Angeles"), // UTC-7/8: previous civil day
		mustLoad(t, "Pacific/Kiritimati"),  // UTC+14: next civil day
	}

	wantStart, wantEnd := contractYearWindow(effectiveFrom, instant.In(zones[0]))
	for _, loc := range zones[1:] {
		start, end := contractYearWindow(effectiveFrom, instant.In(loc))
		if !start.Equal(wantStart) || !end.Equal(wantEnd) {
			t.Errorf("window shifted for host zone %s: got [%s, %s), want [%s, %s)",
				loc, start, end, wantStart, wantEnd)
		}
	}
}

func TestContractYearWindow_AnchorsOnAnniversary(t *testing.T) {
	effectiveFrom := time.Date(2025, 3, 21, 0, 0, 0, 0, time.UTC)
	tehran := domain.BusinessLocation()

	cases := []struct {
		name      string
		now       time.Time
		wantStart time.Time
	}{
		{
			name:      "after this year's anniversary",
			now:       time.Date(2026, 7, 19, 12, 0, 0, 0, tehran),
			wantStart: time.Date(2026, 3, 21, 0, 0, 0, 0, tehran),
		},
		{
			name:      "before this year's anniversary rolls back a year",
			now:       time.Date(2026, 2, 1, 12, 0, 0, 0, tehran),
			wantStart: time.Date(2025, 3, 21, 0, 0, 0, 0, tehran),
		},
		{
			name:      "exactly on the anniversary starts the new window",
			now:       time.Date(2026, 3, 21, 0, 0, 0, 0, tehran),
			wantStart: time.Date(2026, 3, 21, 0, 0, 0, 0, tehran),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := contractYearWindow(effectiveFrom, tc.now)
			if !start.Equal(tc.wantStart) {
				t.Errorf("start = %s, want %s", start, tc.wantStart)
			}
			if !end.Equal(tc.wantStart.AddDate(1, 0, 0)) {
				t.Errorf("end = %s, want %s", end, tc.wantStart.AddDate(1, 0, 0))
			}
			if !start.Before(tc.now.Add(time.Nanosecond)) || !end.After(tc.now) {
				t.Errorf("window [%s, %s) must contain now=%s", start, end, tc.now)
			}
		})
	}
}

func TestBusinessDay_NormalisesToTehranMidnight(t *testing.T) {
	tehran := domain.BusinessLocation()

	// 2026-07-19 21:00 UTC is already 2026-07-20 00:30 in Tehran (+03:30):
	// the civil day differs from the UTC day, which is exactly the case that
	// used to make eligibility depend on the host zone.
	utcEvening := time.Date(2026, 7, 19, 21, 0, 0, 0, time.UTC)
	got := domain.BusinessDay(utcEvening)
	want := time.Date(2026, 7, 20, 0, 0, 0, 0, tehran)
	if !got.Equal(want) {
		t.Errorf("BusinessDay(%s) = %s, want %s", utcEvening, got, want)
	}

	// Same instant in any zone must yield the same business day.
	for _, loc := range []*time.Location{time.UTC, tehran, mustLoad(t, "America/New_York")} {
		if d := domain.BusinessDay(utcEvening.In(loc)); !d.Equal(want) {
			t.Errorf("BusinessDay in %s = %s, want %s", loc, d, want)
		}
	}
}

func TestBusinessLocation_IsTehranOffset(t *testing.T) {
	// Iran abolished DST in 2022, so the offset is a stable +03:30 (12600s)
	// whether tzdata is present or the fixed-zone fallback is used.
	_, offset := time.Date(2026, 7, 19, 12, 0, 0, 0, domain.BusinessLocation()).Zone()
	if offset != 12600 {
		t.Errorf("business offset = %ds, want 12600 (+03:30)", offset)
	}
}

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Skipf("timezone %s unavailable on this host: %v", name, err)
	}
	return loc
}
