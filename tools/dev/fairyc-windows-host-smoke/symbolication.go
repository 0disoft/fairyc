package main

import (
	"fmt"
	"strings"
)

type windowsSymbolicationReceipt struct {
	Name         string
	Displacement uint64
	SourceFile   string
	Line         uint32
}

func validateWindowsSymbolication(receipt windowsSymbolicationReceipt, symbol, sourceBase string, sourceLine uint32) error {
	if receipt.Name != symbol {
		return fmt.Errorf("symbolicator returned %q, want %q", receipt.Name, symbol)
	}
	if receipt.Displacement != 0 {
		return fmt.Errorf("symbolicator returned displacement %d, want zero", receipt.Displacement)
	}
	if sourceLine == 0 || receipt.Line != sourceLine {
		return fmt.Errorf("symbolicator returned source line %d, want %d", receipt.Line, sourceLine)
	}
	normalized := strings.ReplaceAll(receipt.SourceFile, "\\", "/")
	base := normalized
	if separator := strings.LastIndexByte(normalized, '/'); separator >= 0 {
		base = normalized[separator+1:]
	}
	if base != sourceBase {
		return fmt.Errorf("symbolicator returned source basename %q, want %q", base, sourceBase)
	}
	return nil
}
