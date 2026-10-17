package cbackend

import "strings"

// emitter owns only output state and a read-only analyzed plan.
// Each emission gets its own builder, so a completed plan can be emitted again.
type emitter struct {
	analysis *emissionAnalysis
	output   strings.Builder
}
