package cdriver

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWindowsProjectCompileWorkerBound(t *testing.T) {
	for _, tc := range []struct{ cpus, want int }{{0, 1}, {1, 1}, {2, 2}, {128, 2}} {
		if got := windowsProjectCompileWorkers(tc.cpus); got != tc.want {
			t.Fatalf("cpus=%d: workers=%d want %d", tc.cpus, got, tc.want)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	entered, release := make(chan struct{}, 2), make(chan struct{})
	var active, peak, finished atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- compileWindowsProjectOrdered(ctx, 99, 8, func(ctx context.Context, index int) error {
			n := active.Add(1)
			defer active.Add(-1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			if index < 2 {
				entered <- struct{}{}
			}
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
			t.Fatal("two workers did not start")
		}
	}
	close(release)
	if err := <-done; err != nil || peak.Load() != 2 || finished.Load() != 8 || active.Load() != 0 {
		t.Fatalf("compile err=%v peak=%d finished=%d active=%d", err, peak.Load(), finished.Load(), active.Load())
	}
}

func TestWindowsProjectCompileFirstSourceError(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	first, later := errors.New("source 0 failed"), errors.New("source 1 failed")
	laterFinished := make(chan struct{})
	var calls atomic.Int32
	err := compileWindowsProjectOrdered(ctx, 2, 8, func(ctx context.Context, index int) error {
		calls.Add(1)
		if index == 1 {
			close(laterFinished)
			return later
		}
		if index != 0 {
			return errors.New("dispatched after failure")
		}
		select {
		case <-laterFinished:
		case <-ctx.Done():
			return ctx.Err()
		}
		// The lower source must not be cancelled by a later source's failure.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return first
		}
	})
	if err != first || calls.Load() != 2 {
		t.Fatalf("source-order failure: %v calls=%d", err, calls.Load())
	}
}

func TestWindowsProjectCompileCancellationJoins(t *testing.T) {
	for _, fail := range []bool{false, true} {
		ctx, cancel := context.WithCancel(t.Context())
		started := make(chan struct{})
		var finished atomic.Bool
		failure := errors.New("first source failed")
		err := compileWindowsProjectOrdered(ctx, 2, 4, func(ctx context.Context, index int) error {
			if index == 0 {
				<-started
				if fail {
					return failure
				}
				cancel()
				return ctx.Err()
			}
			close(started)
			<-ctx.Done()
			finished.Store(true)
			return ctx.Err()
		})
		want := error(context.Canceled)
		if fail {
			want = failure
		}
		cancel()
		if !errors.Is(err, want) || !finished.Load() {
			t.Fatalf("fail=%t err=%v finished=%t", fail, err, finished.Load())
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := compileWindowsProjectOrdered(ctx, 2, 4, func(context.Context, int) error {
		t.Fatal("started with cancelled parent")
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
