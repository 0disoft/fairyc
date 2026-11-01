//go:build linux

package contractstate

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockLocalState(file *os.File) error {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return ErrPersistenceBusy
	}
	return err
}

func unlockLocalState(file *os.File) { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN) }
