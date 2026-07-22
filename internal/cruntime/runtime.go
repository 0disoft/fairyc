// Package cruntime owns embedded C sources required by the constrained C transport.
package cruntime

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

const (
	trapBasePlaceholder           = "@TRAP_EXIT_BASE@"
	trapReceiptIncludePlaceholder = "@TRAP_RECEIPT_INCLUDE@"
	trapReceiptWritePlaceholder   = "@TRAP_RECEIPT_WRITE@"
	// RunTrapReceiptPrefix identifies the compiler-private native-run trap receipt.
	RunTrapReceiptPrefix = "fairyc-run-trap-v1:"
)

//go:embed stage0_scalar.c
var stage0ScalarTemplate string

//go:embed stage0_drop_trace.c
var stage0DropTrace string

// ScalarSource returns the deterministic Stage 0 scalar runtime with a bounded trap-status base.
func ScalarSource(trapExitBase uint8) (string, error) {
	return renderScalarSource(trapExitBase, false)
}

// WindowsRunSource returns the scalar runtime with one unbuffered compiler-private trap receipt.
// The receipt lets the caller distinguish every normal i32 return from reserved trap exit codes.
func WindowsRunSource(trapExitBase uint8) (string, error) {
	return renderScalarSource(trapExitBase, true)
}

func renderScalarSource(trapExitBase uint8, runReceipt bool) (string, error) {
	if trapExitBase > 248 {
		return "", fmt.Errorf("trap exit base %d leaves no room for seven categories", trapExitBase)
	}
	for _, value := range []struct {
		name        string
		placeholder string
	}{
		{name: "trap-base", placeholder: trapBasePlaceholder},
		{name: "trap-receipt include", placeholder: trapReceiptIncludePlaceholder},
		{name: "trap-receipt write", placeholder: trapReceiptWritePlaceholder},
	} {
		if strings.Count(stage0ScalarTemplate, value.placeholder) != 1 {
			return "", fmt.Errorf("scalar runtime %s placeholder is not unique", value.name)
		}
	}
	include := ""
	write := ""
	if runReceipt {
		include = "#include <io.h>\n"
		write = "    char fy_run_trap_receipt[] = \"" + RunTrapReceiptPrefix + "0\";\n" +
			"    fy_run_trap_receipt[sizeof(fy_run_trap_receipt) - 2] = (char)('0' + category);\n" +
			"    if (_write(2, fy_run_trap_receipt, (unsigned int)(sizeof(fy_run_trap_receipt) - 1)) !=\n" +
			"        (int)(sizeof(fy_run_trap_receipt) - 1)) {\n" +
			"        _Exit(254);\n" +
			"    }\n"
	}
	source := strings.Replace(stage0ScalarTemplate, trapBasePlaceholder, strconv.Itoa(int(trapExitBase)), 1)
	source = strings.Replace(source, trapReceiptIncludePlaceholder, include, 1)
	source = strings.Replace(source, trapReceiptWritePlaceholder, write, 1)
	return source, nil
}

// DropTraceSource returns the scalar runtime plus the Linux-only native-conformance drop recorder.
func DropTraceSource(trapExitBase uint8) (string, error) {
	scalar, err := ScalarSource(trapExitBase)
	if err != nil {
		return "", err
	}
	if stage0DropTrace == "" || strings.Contains(stage0DropTrace, "\r") || !strings.HasSuffix(stage0DropTrace, "\n") {
		return "", fmt.Errorf("drop-trace runtime source is empty or non-canonical")
	}
	return scalar + "\n" + stage0DropTrace, nil
}
