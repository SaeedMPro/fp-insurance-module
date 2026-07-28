package domain

import "testing"

func TestPercentFromFloat_RoundsRatherThanTruncates(t *testing.T) {
	cases := map[float64]Percent{
		0:     0,
		50:    5000,
		70:    7000,
		100:   10000,
		33.33: 3333, // 33.33*100 == 3332.9999999999995 in float64 — must not truncate
		12.5:  1250,
		0.01:  1,
		99.99: 9999,
	}
	for in, want := range cases {
		if got := PercentFromFloat(in); got != want {
			t.Errorf("PercentFromFloat(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestPercentRoundTrip(t *testing.T) {
	for _, f := range []float64{0, 0.01, 12.5, 33.33, 50, 70, 99.99, 100} {
		if got := PercentFromFloat(f).Float(); got != f {
			t.Errorf("round trip %v -> %v", f, got)
		}
	}
}

func TestPercentApplyTo_HalfUpRounding(t *testing.T) {
	cases := []struct {
		name    string
		percent float64
		amount  Rial
		want    Rial
	}{
		{"exact", 70, 400000, 280000},
		{"half rounds up", 50, 1001, 501},          // 500.5 -> 501
		{"below half rounds down", 33, 1001, 330},  // 330.33 -> 330
		{"above half rounds up", 12.5, 8001, 1000}, // 1000.125 -> 1000
		{"one rial half up", 50, 1, 1},             // 0.5 -> 1
		{"third of three", 33.33, 3, 1},            // 0.9999 -> 1
		{"zero percent", 0, 1000000, 0},
		{"full percent", 100, 123456, 123456},
		{"zero amount", 70, 0, 0},
		{"large amount stays exact", 90, 987654321, 888888889}, // 888888888.9 -> 888888889
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PercentFromFloat(tc.percent).ApplyTo(tc.amount); got != tc.want {
				t.Errorf("%v%% of %d = %d, want %d", tc.percent, tc.amount, got, tc.want)
			}
		})
	}
}

func TestPercentApplyTo_NoOverflowAtSchemaCeiling(t *testing.T) {
	// NUMERIC(14,0) ceiling is 99_999_999_999_999.
	const ceiling = Rial(99_999_999_999_999)
	if got := PercentFromFloat(100).ApplyTo(ceiling); got != ceiling {
		t.Errorf("100%% of ceiling = %d, want %d", got, ceiling)
	}
	if got := PercentFromFloat(50).ApplyTo(ceiling); got != 50_000_000_000_000 {
		t.Errorf("50%% of ceiling = %d, want 50000000000000", got)
	}
}

func TestRialFromFloat_RoundsHalfUp(t *testing.T) {
	cases := map[float64]Rial{0: 0, 1: 1, 1.4: 1, 1.5: 2, 2.5: 3, 330.33: 330, 500.5: 501}
	for in, want := range cases {
		if got := RialFromFloat(in); got != want {
			t.Errorf("RialFromFloat(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestRialPointerHelpers(t *testing.T) {
	if RialPtrFromFloatPtr(nil) != nil {
		t.Error("nil float pointer must map to nil Rial pointer")
	}
	if FloatPtrFromRialPtr(nil) != nil {
		t.Error("nil Rial pointer must map to nil float pointer")
	}
	f := 3000000.0
	r := RialPtrFromFloatPtr(&f)
	if r == nil || *r != 3000000 {
		t.Fatalf("got %v, want 3000000", r)
	}
	back := FloatPtrFromRialPtr(r)
	if back == nil || *back != f {
		t.Fatalf("round trip got %v, want %v", back, f)
	}
}
