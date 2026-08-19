package contractabi

import "fmt"

// Diagnostic is one source-relative contract-interface rejection.
type Diagnostic struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s at %d:%d: %s", d.Code, d.Line, d.Column, d.Message)
}
