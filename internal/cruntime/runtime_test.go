package cruntime

import (
	"strings"
	"testing"
)

func TestScalarSourceBindsTrapBase(t *testing.T) {
	t.Parallel()
	source, err := ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(source, trapBasePlaceholder) || strings.Contains(source, trapReceiptIncludePlaceholder) ||
		strings.Contains(source, trapReceiptWritePlaceholder) || strings.Contains(source, RunTrapControlEnvironment) ||
		!strings.Contains(source, "UINT32_C(192)") || !strings.Contains(source, "fy_runtime_trap") {
		t.Fatalf("ScalarSource() = %q", source)
	}
	repeated, err := ScalarSource(192)
	if err != nil || repeated != source || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
		t.Fatalf("repeated ScalarSource() = %q, %v", repeated, err)
	}
}

func TestWindowsRunSourceAddsClosedTrapControlWrite(t *testing.T) {
	t.Parallel()
	source, err := WindowsRunSource(128)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(source, trapBasePlaceholder) || strings.Contains(source, trapReceiptIncludePlaceholder) ||
		strings.Contains(source, trapReceiptWritePlaceholder) || !strings.Contains(source, "#include <windows.h>") ||
		strings.Count(source, RunTrapControlEnvironment) != 1 || !strings.Contains(source, "WriteFile(") ||
		strings.Contains(source, "_write(2,") {
		t.Fatalf("WindowsRunSource() = %q", source)
	}
	repeated, err := WindowsRunSource(128)
	if err != nil || repeated != source || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
		t.Fatalf("repeated WindowsRunSource() = %q, %v", repeated, err)
	}
}

func TestLinuxRunSourceAddsDedicatedTrapControlWrite(t *testing.T) {
	t.Parallel()
	source, err := LinuxRunSource(128)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"#include <errno.h>", "#include <unistd.h>",
		"write(3, &fy_trap_receipt, 1)", "errno == EINTR", "_Exit(254)",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("LinuxRunSource() is missing %q", required)
		}
	}
	if strings.Contains(source, trapBasePlaceholder) || strings.Contains(source, trapReceiptIncludePlaceholder) ||
		strings.Contains(source, trapReceiptWritePlaceholder) || strings.Contains(source, "<windows.h>") ||
		strings.Contains(source, RunTrapControlEnvironment) || strings.Contains(source, "WriteFile(") {
		t.Fatal("LinuxRunSource() contains a placeholder or Windows receipt")
	}
	repeated, err := LinuxRunSource(128)
	if err != nil || repeated != source || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
		t.Fatalf("repeated LinuxRunSource() is not deterministic canonical C: %v", err)
	}
}

func TestScalarSourceRejectsOverflowingBase(t *testing.T) {
	t.Parallel()
	if _, err := ScalarSource(249); err == nil {
		t.Fatal("ScalarSource() accepted a base that overflows process status")
	}
}

func TestDropTraceSourceAddsOnlyTheClosedRecorder(t *testing.T) {
	t.Parallel()
	source, err := DropTraceSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"UINT32_C(192)",
		"void fy_runtime_trace_drop(uint32_t site_id)",
		"write(STDERR_FILENO, record + offset, sizeof record - offset)",
		"_Exit(254)",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("DropTraceSource() is missing %q", required)
		}
	}
	if strings.Count(source, "fy_runtime_trace_drop") != 1 || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
		t.Fatalf("DropTraceSource() is not deterministic canonical C")
	}
}
