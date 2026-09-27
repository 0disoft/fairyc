//go:build !windows

package compiler_test

import "errors"

func pinScannerBuildLowestAllowedCPU() (string, func() error, error) {
	return "", nil, errors.New("build affinity control requires Windows")
}
