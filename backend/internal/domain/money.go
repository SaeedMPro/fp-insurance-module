package domain

import "math"

// Rial is a whole number of Iranian rials. Money never travels through this
// system as a float: binary floating point cannot represent decimal amounts
// exactly, and the rial has no fractional unit in everyday use.
//
// Range: int64 covers ±9.2×10^18 rial, five orders of magnitude beyond the
// schema's own NUMERIC(14,0) ceiling, so overflow is not a practical concern.
type Rial int64

// Percent is a coverage percentage in basis points: 1% = 100 bp, so 70.00%
// is 7000 and 33.33% is 3333. Storing hundredths as an integer keeps the
// pricing arithmetic exact while still matching the schema's NUMERIC(5,2).
type Percent int32

// PercentFromFloat converts a NUMERIC(5,2) percentage (as read from the
// database or an API request) to basis points. Rounding — rather than
// truncation — is essential: 33.33 × 100 is 3332.9999999999995 in float64,
// which would truncate to 3332 and quietly underpay every claim.
func PercentFromFloat(f float64) Percent {
	return Percent(math.Round(f * 100))
}

// Float renders basis points back as a percentage for the wire format and the
// database, where the contract is a NUMERIC(5,2)/JSON number.
func (p Percent) Float() float64 { return float64(p) / 100 }

// ApplyTo returns p percent of amount, rounded half-up to the whole rial.
// This is the ONLY place a monetary rounding decision is made;
// every other calculation is exact integer arithmetic.
func (p Percent) ApplyTo(amount Rial) Rial {
	if amount <= 0 || p <= 0 {
		return 0
	}
	// amount × bp / 10_000, rounded half-up. The intermediate product peaks at
	// ~10^18 for schema-legal amounts, comfortably inside int64.
	return Rial((int64(amount)*int64(p) + 5_000) / 10_000)
}

// RialFromFloat converts a database NUMERIC or JSON number to Rial, rounding
// half-up. Used only at the storage/transport boundary while legacy
// fractional values may still exist; the migration to NUMERIC(14,0) makes
// every stored value integral.
func RialFromFloat(f float64) Rial {
	return Rial(math.Round(f))
}

// Float renders a Rial for the wire format and the database.
func (r Rial) Float() float64 { return float64(r) }

// RialPtrFromFloatPtr maps an optional NUMERIC (nil = "no cap configured").
func RialPtrFromFloatPtr(f *float64) *Rial {
	if f == nil {
		return nil
	}
	r := RialFromFloat(*f)
	return &r
}

// FloatPtrFromRialPtr is the inverse, for storage writes and DTOs.
func FloatPtrFromRialPtr(r *Rial) *float64 {
	if r == nil {
		return nil
	}
	f := r.Float()
	return &f
}
