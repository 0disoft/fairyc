package cdriver

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestOwnedProcessGroupCompletionOrderAndErrors(t *testing.T) {
	observeFailure := errors.New("observe failed")
	cleanupFailure := errors.New("terminate failed")
	exitFailure := errors.New("exit status 23")
	for _, test := range []struct {
		name                     string
		observe, terminate, wait error
	}{
		{"success", nil, nil, nil},
		{"ordinary exit", nil, nil, exitFailure},
		{"already gone", nil, os.ErrProcessDone, nil},
		{"observation failure", observeFailure, nil, exitFailure},
		{"cleanup failure", nil, cleanupFailure, exitFailure},
	} {
		t.Run(test.name, func(t *testing.T) {
			var order []string
			runErr, cleanupErr := completeOwnedProcessGroup(
				func() error { order = append(order, "observe"); return test.observe },
				func() error { order = append(order, "terminate"); return test.terminate },
				func() error { order = append(order, "wait"); return test.wait },
			)
			if !reflect.DeepEqual(order, []string{"observe", "terminate", "wait"}) || runErr != test.wait {
				t.Fatalf("order=%v run error=%v", order, runErr)
			}
			wantCleanup := test.observe != nil || test.terminate != nil && test.terminate != os.ErrProcessDone
			if (cleanupErr != nil) != wantCleanup || test.observe != nil && !errors.Is(cleanupErr, test.observe) || test.terminate == cleanupFailure && !errors.Is(cleanupErr, cleanupFailure) {
				t.Fatalf("cleanup error lost: %v", cleanupErr)
			}
		})
	}
}
