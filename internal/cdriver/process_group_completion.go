package cdriver

import (
	"errors"
	"os"
)

// Observation must keep the leader unreaped until termination, reserving its
// PID/process-group identity. Always perform cleanup and the final wait, and
// keep cleanup errors separate from an ordinary process exit status.
func completeOwnedProcessGroup(observe, terminate, wait func() error) (runErr, cleanupErr error) {
	observeErr := observe()
	terminateErr := terminate()
	if errors.Is(terminateErr, os.ErrProcessDone) {
		terminateErr = nil
	}
	runErr = wait()
	return runErr, errors.Join(observeErr, terminateErr)
}
