package source

import "testing"

func TestWorkBudgetRejectsOverflowAndNegativeCharges(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, charges := range [][]int{{maxInt, 1}, {1, maxInt}, {-1}} {
		limits := DefaultLimits()
		limits.MaxWorkUnits = maxInt
		budget := NewBudget(limits, 0)
		for i, charge := range charges {
			got := budget.ChargeWork(charge)
			if got != (i < len(charges)-1) {
				t.Fatalf("charge %d: success=%v", charge, got)
			}
		}
		if !budget.Exhausted() || budget.ChargeWork(0) {
			t.Fatal("exhaustion must remain sticky")
		}
	}
}
