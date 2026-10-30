package cdriver

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"
)

// PartitionBuildStats records admission for one C build, not total process RSS.
// Successful artifacts carry it; failures after admission expose it via
// PartitionBuildError. Normal CLI stdout/stderr stays unchanged on success.
type PartitionBuildStats struct {
	Workers                  int
	GOMAXPROCS               int
	AvailableMemoryBytes     uint64
	GoHeapAllocBeforeBytes   uint64
	GoHeapAllocAfterBytes    uint64
	GoHeapReleasedAfterBytes uint64
	MemoryPreparation        time.Duration
}

type PartitionBuildError struct {
	Stats PartitionBuildStats
	Err   error
}

func (e *PartitionBuildError) Error() string {
	s := e.Stats
	return fmt.Sprintf("partitioned C build (workers=%d gomaxprocs=%d available-bytes=%d go-heap-before=%d go-heap-after=%d go-released=%d memory-preparation=%s): %v",
		s.Workers, s.GOMAXPROCS, s.AvailableMemoryBytes, s.GoHeapAllocBeforeBytes, s.GoHeapAllocAfterBytes, s.GoHeapReleasedAfterBytes, s.MemoryPreparation, e.Err)
}

func (e *PartitionBuildError) Unwrap() error { return e.Err }

func preparePartitionMemory(sourceBytes int) PartitionBuildStats {
	var stats PartitionBuildStats
	// Only large generated inputs justify an explicit collection/scavenge.
	// By this boundary CLI workspace frontend/emitter frames have returned. A collection
	// alone can leave unused Go heap charged to memory.current, so return pages
	// to the OS before sampling the container's remaining compiler budget.
	if sourceBytes > 2<<20 {
		start := time.Now()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		debug.FreeOSMemory()
		runtime.ReadMemStats(&after)
		stats.GoHeapAllocBeforeBytes = before.HeapAlloc
		stats.GoHeapAllocAfterBytes = after.HeapAlloc
		stats.GoHeapReleasedAfterBytes = after.HeapReleased
		stats.MemoryPreparation = time.Since(start)
	}
	return stats
}
