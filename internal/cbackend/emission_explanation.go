package cbackend

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

type EmissionStep struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
}

type EmissionExplanation struct {
	Schema  string         `json:"schema"`
	Entry   string         `json:"entry"`
	Symbol  string         `json:"symbol"`
	Emitted bool           `json:"emitted"`
	Path    []EmissionStep `json:"path"`
}

type EmissionSelectorError struct{ Symbol string }

func (err EmissionSelectorError) Error() string {
	return fmt.Sprintf("expected one internal function matching %q", err.Symbol)
}

// EmitWithExplanation observes the same discovery edges used by Emit.
// The first discovery of a function records exactly one predecessor.
func EmitWithExplanation(program mir.Program, entry string, profile Profile, selector string) (Unit, EmissionExplanation, error) {
	var selected *mir.Function
	qualified := strings.Contains(selector, "::")
	for i := range program.Functions {
		function := &program.Functions[i]
		name := function.Name
		if index := strings.LastIndex(name, "::"); !qualified && index >= 0 {
			name = name[index+len("::"):]
		}
		if name == selector {
			if selected != nil {
				return Unit{}, EmissionExplanation{}, EmissionSelectorError{selector}
			}
			selected = function
		}
	}
	if selected == nil {
		return Unit{}, EmissionExplanation{}, EmissionSelectorError{selector}
	}
	predecessors := make(map[hir.SymbolID]emissionPredecessor)
	plan, err := analyzeEmissionTrace(context.Background(), program, entry, profile, false, func(from, to *mir.Function, reason string) {
		predecessors[to.Symbol] = emissionPredecessor{from.Symbol, EmissionStep{From: from.Name, To: to.Name, Reason: reason}}
	})
	if err != nil {
		return Unit{}, EmissionExplanation{}, err
	}
	unit, err := (&emitter{analysis: plan}).emitUnit()
	if err != nil {
		return Unit{}, EmissionExplanation{}, err
	}
	return unit, explainEmissionPath(entry, selected, predecessors), nil
}

type emissionPredecessor struct {
	symbol hir.SymbolID
	step   EmissionStep
}

func explainEmissionPath(entry string, function *mir.Function, predecessors map[hir.SymbolID]emissionPredecessor) EmissionExplanation {
	_, found := predecessors[function.Symbol]
	report := EmissionExplanation{
		Schema: "fairyc.why-emitted/v1", Entry: entry, Symbol: function.Name,
		Emitted: function.Name == entry || found, Path: []EmissionStep{},
	}
	for previous, ok := predecessors[function.Symbol]; ok; previous, ok = predecessors[previous.symbol] {
		report.Path = append(report.Path, previous.step)
	}
	slices.Reverse(report.Path)
	return report
}
