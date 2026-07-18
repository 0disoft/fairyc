// Package cruntime owns embedded C sources required by the constrained C transport.
package cruntime

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

const trapBasePlaceholder = "@TRAP_EXIT_BASE@"

//go:embed stage0_scalar.c
var stage0ScalarTemplate string

// ScalarSource returns the deterministic Stage 0 scalar runtime with a bounded trap-status base.
func ScalarSource(trapExitBase uint8) (string, error) {
	if trapExitBase > 248 {
		return "", fmt.Errorf("trap exit base %d leaves no room for seven categories", trapExitBase)
	}
	if strings.Count(stage0ScalarTemplate, trapBasePlaceholder) != 1 {
		return "", fmt.Errorf("scalar runtime trap-base placeholder is not unique")
	}
	return strings.Replace(stage0ScalarTemplate, trapBasePlaceholder, strconv.Itoa(int(trapExitBase)), 1), nil
}
