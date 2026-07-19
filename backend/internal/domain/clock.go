package domain

import "time"

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
