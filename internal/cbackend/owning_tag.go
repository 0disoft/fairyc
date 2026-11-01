package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
)

// Bound recursive ownership transport and emitted cleanup size. Existing inert
// tagged families retain their own admission rules.
const maxOwningTagDepth = 8

func (e *emitter) emitOwnedDropGlue(output *strings.Builder, typ hir.Type, expression, condition string, span hir.Span) error {
	if _, direct := e.analysis.resourceByType[typ]; direct {
		return e.emitDirectDropGlue(output, typ, expression, condition, span)
	}
	tagged, found := e.analysis.taggedType(typ)
	if !found || !tagged.needsDrop {
		return nil
	}
	if condition != "" {
		fmt.Fprintf(output, "    if (%s) {\n", condition)
		if err := e.emitOwnedDropGlue(output, typ, expression, "", span); err != nil {
			return err
		}
		output.WriteString("    }\n")
		return nil
	}
	for i, payload := range tagged.payloads {
		if !e.analysis.supportedNeedsDropType(payload) {
			continue
		}
		tag := "FY_OPTION_SOME"
		if tagged.kind == taggedResult {
			tag = "FY_RESULT_OK"
			if i == 1 {
				tag = "FY_RESULT_ERR"
			}
		}
		if err := e.emitOwnedDropGlue(output, payload, fmt.Sprintf("(%s).p_%d", expression, i), fmt.Sprintf("%s.tag == %s", expression, tag), span); err != nil {
			return err
		}
	}
	return nil
}

func (e *emitter) emitOwnedDropTrace(output *strings.Builder, typ hir.Type, expression, condition string, span hir.Span) error {
	if _, direct := e.analysis.resourceByType[typ]; direct {
		return e.emitDropTraceCall(output, typ, span, condition)
	}
	tagged, found := e.analysis.taggedType(typ)
	if !found || !tagged.needsDrop {
		return nil
	}
	for i, payload := range tagged.payloads {
		tag := "FY_OPTION_SOME"
		if tagged.kind == taggedResult {
			tag = "FY_RESULT_OK"
			if i == 1 {
				tag = "FY_RESULT_ERR"
			}
		}
		active := fmt.Sprintf("%s.tag == %s", expression, tag)
		if condition != "" {
			active = "(" + condition + ") && (" + active + ")"
		}
		if err := e.emitOwnedDropTrace(output, payload, fmt.Sprintf("(%s).p_%d", expression, i), active, span); err != nil {
			return err
		}
	}
	return nil
}
