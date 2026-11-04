package cbackend

import "strings"

// emitter owns only output state and a read-only analyzed plan.
// Each emission gets its own builder, so a completed plan can be emitted again.
type emitter struct {
	analysis      *emissionAnalysis
	output        strings.Builder
	partitioned   bool
	dropHelpers   []dropSiteKey
	dropHelperIDs map[dropSiteKey]int
}

func (e *emitter) functionLinkage() string {
	if e.partitioned {
		return ""
	}
	return "static "
}

func (e *emitter) contextError() error {
	if e.analysis.ctx == nil {
		return nil
	}
	return e.analysis.ctx.Err()
}
