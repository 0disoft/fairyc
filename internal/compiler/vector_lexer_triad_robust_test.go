package compiler_test

import "testing"

func TestScannerTriadRobustOrderStatisticRanks(t *testing.T) {
	sorted := make([]float64, 24)
	for i := range sorted {
		sorted[i] = 0.99 + float64(i+1)/1000
	}
	lower, upper := scannerTriadRobustNullBounds(sorted)
	if lower != sorted[6] || upper != sorted[17] || scannerTriadRobustRuntimeUpper(sorted) != sorted[16] {
		t.Fatalf("unexpected order-statistic ranks: lower=%g upper=%g runtime=%g", lower, upper, scannerTriadRobustRuntimeUpper(sorted))
	}
	if lower < 0.98 || upper > 1.02 {
		t.Fatal("central null band should fit the predeclared bias limit")
	}
	for i := 17; i < len(sorted); i++ {
		sorted[i] = 1.021 + float64(i-17)/1000
	}
	if _, changedUpper := scannerTriadRobustNullBounds(sorted); changedUpper <= 1.02 {
		t.Fatal("central null band failed to detect a bias-bound crossing")
	}
	for i := 16; i < len(sorted); i++ {
		sorted[i] = 1.081 + float64(i-16)/1000
	}
	if bound := scannerTriadRobustRuntimeUpper(sorted); bound <= 1.08 {
		t.Fatal("runtime upper bound failed to detect an 8% crossing")
	}
}
