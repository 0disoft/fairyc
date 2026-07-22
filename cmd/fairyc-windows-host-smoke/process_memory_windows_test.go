//go:build windows

package main

import (
	"os"
	"testing"
)

func TestProcessMemoryProbeReadsPositiveWindowsPeaks(t *testing.T) {
	probe, err := openProcessMemoryProbe(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer probe.close()
	workingSet, pagefile, err := probe.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if workingSet == 0 || pagefile == 0 {
		t.Fatalf("process memory peaks = working set %d, pagefile %d", workingSet, pagefile)
	}
}
