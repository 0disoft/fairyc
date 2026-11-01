//go:build !windows && !linux

package contractstate

import (
	"fmt"
	"os"
)

func lockLocalState(*os.File) error {
	return fmt.Errorf("local state persistence is available only on Windows and Linux")
}
func unlockLocalState(*os.File) {}
