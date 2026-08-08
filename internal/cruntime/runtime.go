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
	// RunTrapControlEnvironment carries the inherited Windows handle for the
	// compiler-private native-run trap control channel.
	RunTrapControlEnvironment = "FAIRYC_TRAP_CONTROL_HANDLE_V1"
)

//go:embed stage0_scalar.c
var stage0ScalarTemplate string

//go:embed stage0_drop_trace.c
var stage0DropTrace string

// ScalarSource returns the deterministic Stage 0 scalar runtime with a bounded trap-status base.
func ScalarSource(trapExitBase uint8) (string, error) {
	return renderScalarSource(trapExitBase, false)
}

// WindowsRunSource returns the scalar runtime with one compiler-private trap control write.
// The dedicated inherited channel lets the caller distinguish every normal i32 return from
// reserved trap exit codes without interpreting program stdout or stderr.
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
		include = "#include <windows.h>\n"
		write = "    char fy_trap_handle_text[32];\n" +
			"    DWORD fy_trap_handle_length = GetEnvironmentVariableA(\"" + RunTrapControlEnvironment + "\", fy_trap_handle_text, (DWORD)sizeof(fy_trap_handle_text));\n" +
			"    uintptr_t fy_trap_handle_value = (uintptr_t)0;\n" +
			"    if (fy_trap_handle_length == (DWORD)0 || fy_trap_handle_length >= (DWORD)sizeof(fy_trap_handle_text)) {\n" +
			"        _Exit(254);\n" +
			"    }\n" +
			"    for (DWORD fy_index = (DWORD)0; fy_index < fy_trap_handle_length; ++fy_index) {\n" +
			"        unsigned int fy_digit = (unsigned int)(unsigned char)fy_trap_handle_text[fy_index] - (unsigned int)'0';\n" +
			"        if (fy_digit > UINT32_C(9) || fy_trap_handle_value > (UINTPTR_MAX - (uintptr_t)fy_digit) / (uintptr_t)10) {\n" +
			"            _Exit(254);\n" +
			"        }\n" +
			"        fy_trap_handle_value = fy_trap_handle_value * (uintptr_t)10 + (uintptr_t)fy_digit;\n" +
			"    }\n" +
			"    unsigned char fy_trap_receipt = (unsigned char)category;\n" +
			"    DWORD fy_trap_written = (DWORD)0;\n" +
			"    if (fy_trap_handle_value == (uintptr_t)0 || !WriteFile((HANDLE)fy_trap_handle_value, &fy_trap_receipt, (DWORD)1, &fy_trap_written, NULL) || fy_trap_written != (DWORD)1) {\n" +
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
