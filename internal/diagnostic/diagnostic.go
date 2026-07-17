// Package diagnostic defines source-stable diagnostics shared by Stage 0 frontend packages.
package diagnostic

import "cmp"

// Span is a zero-based, half-open byte range over the original source file.
type Span struct {
	StartByte int
	EndByte   int
}

// Valid reports whether the span is ordered and contained by a source of size bytes.
func (s Span) Valid(size int) bool {
	return s.StartByte >= 0 && s.StartByte <= s.EndByte && s.EndByte <= size
}

// Repair identifies the recovery action that made parser progress.
type Repair string

const (
	RepairNone        Repair = "none"
	RepairInsert      Repair = "insert"
	RepairSkip        Repair = "skip"
	RepairSynchronize Repair = "synchronize"
	RepairExit        Repair = "exit"
)

// Diagnostic is the stable machine-facing part of a frontend error.
// Human wording is deliberately not part of this type yet.
type Diagnostic struct {
	Code   string
	Span   Span
	Repair Repair
	Token  string
}

// Compare orders diagnostics deterministically by source position and stable identity.
func Compare(a, b Diagnostic) int {
	if n := cmp.Compare(a.Span.StartByte, b.Span.StartByte); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Span.EndByte, b.Span.EndByte); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Code, b.Code); n != 0 {
		return n
	}
	if n := cmp.Compare(string(a.Repair), string(b.Repair)); n != 0 {
		return n
	}
	return cmp.Compare(a.Token, b.Token)
}
