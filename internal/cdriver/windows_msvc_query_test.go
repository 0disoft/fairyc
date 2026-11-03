package cdriver

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"
)

func TestMSVCInstallationQueryRetriesOnlyWaitDelay(t *testing.T) {
	for _, firstErr := range []error{exec.ErrWaitDelay, fmt.Errorf("wrapped: %w", exec.ErrWaitDelay)} {
		calls := 0
		out, err := queryMSVCInstallation(time.Second, func(remaining time.Duration) (string, error) {
			calls++
			if remaining <= 0 || remaining > time.Second {
				t.Fatalf("invalid remaining budget: %s", remaining)
			}
			if calls == 1 {
				return "partial path must not be used", firstErr
			}
			return "fresh complete path", nil
		})
		if err != nil || out != "fresh complete path" || calls != 2 {
			t.Fatalf("query recovery: output=%q err=%v calls=%d", out, err, calls)
		}
	}
}

func TestMSVCInstallationQueryDoesNotRetryOtherResults(t *testing.T) {
	for _, want := range []error{nil, &exec.ExitError{}, context.DeadlineExceeded, context.Canceled, errors.New("output truncated")} {
		calls := 0
		out, err := queryMSVCInstallation(time.Second, func(time.Duration) (string, error) {
			calls++
			return "original", want
		})
		if calls != 1 || err != want || out != "original" {
			t.Fatalf("unexpected retry: err=%v want=%v calls=%d", err, want, calls)
		}
	}
}

func TestMSVCInstallationQueryRetryStillFailsClosed(t *testing.T) {
	calls := 0
	out, err := queryMSVCInstallation(time.Second, func(time.Duration) (string, error) {
		calls++
		return "still incomplete", exec.ErrWaitDelay
	})
	if !errors.Is(err, exec.ErrWaitDelay) || calls != 2 {
		t.Fatalf("retry must not hide incomplete output: output=%q err=%v calls=%d", out, err, calls)
	}
}

func TestMSVCInstallationQuerySharesDeadline(t *testing.T) {
	now := time.Unix(100, 0)
	calls := 0
	out, err := queryMSVCInstallation(time.Millisecond, func(time.Duration) (string, error) {
		calls++
		now = now.Add(time.Millisecond)
		return "incomplete", exec.ErrWaitDelay
	}, msvcQueryOptions{Now: func() time.Time { return now }})
	if !errors.Is(err, exec.ErrWaitDelay) || !errors.Is(err, context.DeadlineExceeded) || calls != 1 || out != "" {
		t.Fatalf("query exceeded shared deadline: out=%q err=%v calls=%d", out, err, calls)
	}
}

func TestMSVCInstallationQueryDeadlineBoundary(t *testing.T) {
	const budget = time.Second
	for _, attempt := range []int{1, 2} {
		for _, elapsed := range []time.Duration{budget - time.Nanosecond, budget, budget + time.Nanosecond} {
			t.Run(fmt.Sprintf("attempt-%d/%s", attempt, elapsed), func(t *testing.T) {
				now := time.Unix(100, 0)
				calls := 0
				out, err := queryMSVCInstallation(budget, func(time.Duration) (string, error) {
					calls++
					if calls < attempt {
						return "partial", exec.ErrWaitDelay
					}
					now = now.Add(elapsed)
					return "complete", nil
				}, msvcQueryOptions{Now: func() time.Time { return now }})
				if calls != attempt {
					t.Fatalf("calls=%d, want %d", calls, attempt)
				}
				if elapsed < budget {
					if err != nil || out != "complete" {
						t.Fatalf("on-time result rejected: out=%q err=%v", out, err)
					}
				} else if !errors.Is(err, context.DeadlineExceeded) || out != "" {
					t.Fatalf("late result accepted: out=%q err=%v", out, err)
				}
			})
		}
	}
}

func TestMSVCInstallationQueryInvalidBudgetDoesNotPerformIO(t *testing.T) {
	for _, budget := range []time.Duration{-time.Nanosecond, 0} {
		t.Run(budget.String(), func(t *testing.T) {
			out, err := queryMSVCInstallation(budget, func(time.Duration) (string, error) {
				t.Fatal("exhausted budget performed I/O")
				return "", nil
			})
			if !errors.Is(err, context.DeadlineExceeded) || out != "" {
				t.Fatalf("out=%q err=%v", out, err)
			}
		})
	}
}

func TestMSVCInstallationQueryRetryGetsOnlyRemainingNanosecond(t *testing.T) {
	now := time.Unix(100, 0)
	calls := 0
	out, err := queryMSVCInstallation(time.Second, func(remaining time.Duration) (string, error) {
		calls++
		if calls == 1 {
			if remaining != time.Second {
				t.Fatalf("initial timeout=%s", remaining)
			}
			now = now.Add(time.Second - time.Nanosecond)
			return "partial", exec.ErrWaitDelay
		}
		if remaining != time.Nanosecond {
			t.Fatalf("retry timeout=%s, want 1ns", remaining)
		}
		return "complete", nil
	}, msvcQueryOptions{Now: func() time.Time { return now }})
	if err != nil || out != "complete" || calls != 2 {
		t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
	}
}

func TestMSVCInstallationQueryCancellation(t *testing.T) {
	for _, cancelAt := range []int{0, 1, 2} {
		for _, queryErr := range []error{nil, exec.ErrWaitDelay} {
			t.Run(fmt.Sprintf("cancel-at-%d/err-%v", cancelAt, queryErr), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if cancelAt == 0 {
					cancel()
				}
				calls := 0
				out, err := queryMSVCInstallation(time.Second, func(time.Duration) (string, error) {
					calls++
					if calls < cancelAt {
						return "partial", exec.ErrWaitDelay
					}
					cancel()
					return "must discard", queryErr
				}, msvcQueryOptions{Context: ctx})
				if calls != cancelAt || !errors.Is(err, context.Canceled) || out != "" {
					t.Fatalf("out=%q err=%v calls=%d, want %d", out, err, calls, cancelAt)
				}
				if cancelAt > 0 && queryErr != nil && !errors.Is(err, queryErr) {
					t.Fatalf("cancellation lost query error: %v", err)
				}
			})
		}
	}
}
