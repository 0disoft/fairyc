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
	calls := 0
	_, err := queryMSVCInstallation(time.Millisecond, func(time.Duration) (string, error) {
		calls++
		time.Sleep(10 * time.Millisecond)
		return "incomplete", exec.ErrWaitDelay
	})
	if !errors.Is(err, exec.ErrWaitDelay) || calls != 1 {
		t.Fatalf("query exceeded shared deadline: err=%v calls=%d", err, calls)
	}
}
