package cdriver

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

func TestPartitionMemoryPreparation(t *testing.T) {
	if stats := preparePartitionMemory(1024); stats != (PartitionBuildStats{}) {
		t.Fatalf("small build unexpectedly collected memory: %+v", stats)
	}
	// Disable automatic collection only for this nonparallel test, making the
	// dead allocation remain charged until build preparation explicitly runs.
	previous := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previous)
	leaveDeadFrontendAllocation()
	stats := preparePartitionMemory(3 << 20)
	if stats.GoHeapAllocBeforeBytes < stats.GoHeapAllocAfterBytes+16<<20 || stats.MemoryPreparation <= 0 {
		t.Fatalf("large build did not collect dead frontend memory: %+v", stats)
	}
	t.Logf("partition memory preparation: %+v", stats)
}

//go:noinline
func leaveDeadFrontendAllocation() {
	data := make([]byte, 32<<20)
	for i := 0; i < len(data); i += 4096 {
		data[i] = 1
	}
	runtime.KeepAlive(data)
}

func TestPartitionBuildErrorPreservesCauseAndAdmission(t *testing.T) {
	failure := &PartitionBuildError{
		Stats: PartitionBuildStats{Workers: 1, GOMAXPROCS: 4, AvailableMemoryBytes: 618 << 20},
		Err:   context.DeadlineExceeded,
	}
	err := fmt.Errorf("build failed: %w", failure)
	var found *PartitionBuildError
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &found) || found.Stats != failure.Stats {
		t.Fatalf("partition failure lost cause/admission: %v", err)
	}
	if !strings.Contains(err.Error(), "workers=1 gomaxprocs=4 available-bytes=648019968") {
		t.Fatalf("build failure omitted worker decision: %v", err)
	}
}
