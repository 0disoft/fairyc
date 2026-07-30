package main

import (
	"strings"
	"testing"
)

func TestValidateWindowsSymbolicationAcceptsExactFunctionAndSource(t *testing.T) {
	receipt := windowsSymbolicationReceipt{Name: "fy_rel32_source", SourceFile: `C:\temp\caller.c`, Line: 8}
	if err := validateWindowsSymbolication(receipt, "fy_rel32_source", "caller.c", 8); err != nil {
		t.Fatal(err)
	}
}

func TestValidateWindowsSymbolicationRejectsDrift(t *testing.T) {
	tests := []struct {
		name    string
		receipt windowsSymbolicationReceipt
		want    string
	}{
		{name: "symbol", receipt: windowsSymbolicationReceipt{Name: "other", SourceFile: "caller.c", Line: 8}, want: "want"},
		{name: "displacement", receipt: windowsSymbolicationReceipt{Name: "fy_rel32_source", Displacement: 1, SourceFile: "caller.c", Line: 8}, want: "displacement"},
		{name: "line", receipt: windowsSymbolicationReceipt{Name: "fy_rel32_source", SourceFile: "caller.c"}, want: "source line"},
		{name: "wrong nonzero line", receipt: windowsSymbolicationReceipt{Name: "fy_rel32_source", SourceFile: "caller.c", Line: 9}, want: "want 8"},
		{name: "file", receipt: windowsSymbolicationReceipt{Name: "fy_rel32_source", SourceFile: "target.c", Line: 8}, want: "source basename"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validateWindowsSymbolication(testCase.receipt, "fy_rel32_source", "caller.c", 8); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("validateWindowsSymbolication() error = %v, want %q", err, testCase.want)
			}
		})
	}
}
