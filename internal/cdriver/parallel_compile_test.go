package cdriver

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallelCompileBoundAndCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	entered, release := make(chan struct{}, 4), make(chan struct{})
	var active, peak, finished atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- compileParallel(ctx, 2, 4, func(ctx context.Context, index int) error {
			n := active.Add(1)
			defer active.Add(-1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			entered <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			finished.Add(1)
			return nil
		})
	}()
	for range 2 {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("jobs did not run concurrently")
		}
	}
	if active.Load() != 2 {
		t.Fatal("parallel worker bound lost")
	}
	close(release)
	if err := <-done; err != nil || peak.Load() != 2 || finished.Load() != 4 {
		t.Fatalf("workers: %v peak=%d finished=%d", err, peak.Load(), finished.Load())
	}

	failure := errors.New("compiler failed")
	started := make(chan struct{})
	var cleaned atomic.Bool
	err := compileParallel(ctx, 2, 2, func(ctx context.Context, index int) error {
		if index == 0 {
			<-started
			return failure
		}
		close(started)
		<-ctx.Done()
		cleaned.Store(true)
		return ctx.Err()
	})
	if err != failure || !cleaned.Load() {
		t.Fatalf("did not cancel and join sibling: %v", err)
	}
}

func TestParallelCompileSharesDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	var calls atomic.Int32
	err := compileParallel(ctx, 1, 5, func(ctx context.Context, index int) error {
		if got, ok := ctx.Deadline(); !ok || got != deadline {
			return errors.New("worker renewed the build deadline")
		}
		calls.Add(1)
		cancel()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("new deadline or job after cancellation: %v calls=%d", err, calls.Load())
	}
}

func TestCompileWorkerResourceCaps(t *testing.T) {
	for _, test := range []struct {
		cpu    int
		memory uint64
		want   int
	}{
		{8, 0, 1}, {8, 256 << 20, 1}, {8, 1 << 30, 1}, {8, 4 << 30, 2}, {1, 4 << 30, 1},
		{4, (1280 << 20) - 1, 1}, {4, 1280 << 20, 2}, {0, 4 << 30, 1},
	} {
		if got := linuxCompileWorkers(test.cpu, test.memory); got != test.want {
			t.Fatalf("cpu=%d memory=%d: %d want %d", test.cpu, test.memory, got, test.want)
		}
	}
	if err := validateCPartitions(CPartitions{Header: "h", Sources: []string{"a", "b"}}, "runtime"); err != nil {
		t.Fatal(err)
	}
	if err := validateCPartitions(CPartitions{Header: "h", Sources: []string{""}}, "runtime"); err == nil {
		t.Fatal("empty partition admitted")
	}
}
