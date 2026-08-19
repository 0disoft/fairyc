package chainprimitives

import (
	"errors"
	"math"
	"testing"
)

func TestU64OverflowPolicies(t *testing.T) {
	t.Parallel()

	if _, err := AddU64Checked(math.MaxUint64, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("checked add error = %v, want ErrOverflow", err)
	}
	if got := AddU64Wrapping(math.MaxUint64, 1); got != 0 {
		t.Fatalf("wrapping add = %d, want 0", got)
	}
	if got := AddU64Saturating(math.MaxUint64, 1); got != math.MaxUint64 {
		t.Fatalf("saturating add = %d, want MaxUint64", got)
	}

	if _, err := SubU64Checked(0, 1); !errors.Is(err, ErrUnderflow) {
		t.Fatalf("checked sub error = %v, want ErrUnderflow", err)
	}
	if got := SubU64Wrapping(0, 1); got != math.MaxUint64 {
		t.Fatalf("wrapping sub = %d, want MaxUint64", got)
	}
	if got := SubU64Saturating(0, 1); got != 0 {
		t.Fatalf("saturating sub = %d, want 0", got)
	}

	if _, err := MulU64Checked(math.MaxUint64, 2); !errors.Is(err, ErrOverflow) {
		t.Fatalf("checked mul error = %v, want ErrOverflow", err)
	}
	if got := MulU64Saturating(math.MaxUint64, 2); got != math.MaxUint64 {
		t.Fatalf("saturating mul = %d, want MaxUint64", got)
	}
}
