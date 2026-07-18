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
	if strings.Contains(source, trapBasePlaceholder) || !strings.Contains(source, "UINT32_C(192)") || !strings.Contains(source, "fy_runtime_trap") {
		t.Fatalf("ScalarSource() = %q", source)
	}
	repeated, err := ScalarSource(192)
	if err != nil || repeated != source || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
		t.Fatalf("repeated ScalarSource() = %q, %v", repeated, err)
	}
}

func TestScalarSourceRejectsOverflowingBase(t *testing.T) {
	t.Parallel()
	if _, err := ScalarSource(249); err == nil {
		t.Fatal("ScalarSource() accepted a base that overflows process status")
	}
}
