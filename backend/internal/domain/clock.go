package domain

import (
	"sync"
	"time"
)

// Clock abstracts "now" so services with time-dependent business rules
// (waiting periods, contract-year windows, transition timestamps) are
// deterministic under test. Production wiring uses SystemClock.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// FixedClock always returns the same instant — for tests.
type FixedClock struct{ T time.Time }

func (c FixedClock) Now() time.Time { return c.T }

// businessLocationName is the organisation's civil timezone. Every *day
// boundary* decision (contract-year windows, waiting-period eligibility,
// effective dates) is evaluated here rather than in the host's local zone, so
// pricing does not depend on where the server happens to run.
const businessLocationName = "Asia/Tehran"

var (
	businessLocationOnce sync.Once
	businessLocation     *time.Location
)

// BusinessLocation returns the timezone business days are measured in. If the
// host has no tzdata for it, it falls back to Iran's fixed +03:30 offset
// (Iran abolished DST in 2022, so the offset is stable) rather than silently
// reverting to UTC and shifting every day boundary by 3.5 hours.
func BusinessLocation() *time.Location {
	businessLocationOnce.Do(func() {
		if loc, err := time.LoadLocation(businessLocationName); err == nil {
			businessLocation = loc
			return
		}
		businessLocation = time.FixedZone("+0330", int((3*time.Hour + 30*time.Minute).Seconds()))
	})
	return businessLocation
}

// BusinessDay truncates t to midnight in the business timezone. Comparing
// BusinessDay values answers "is this the same/earlier civil day?" without
// being thrown off by the instant's own zone.
func BusinessDay(t time.Time) time.Time {
	loc := BusinessLocation()
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}
