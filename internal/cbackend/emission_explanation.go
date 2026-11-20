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
	return emitWithExplanation(program, entry, profile, selector, false)
}

// EmitStorageTransfersWithExplanation observes the SDK storage-enabled emitter.
func EmitStorageTransfersWithExplanation(program mir.Program, entry string, profile Profile, selector string) (Unit, EmissionExplanation, error) {
	return emitWithExplanation(program, entry, profile, selector, true)
}

func emitWithExplanation(program mir.Program, entry string, profile Profile, selector string, storageTransfers bool) (Unit, EmissionExplanation, error) {
	var selected *mir.Function
	qualified := strings.Contains(selector, "::")
	for i := range program.Functions {
		function := &program.Functions[i]
		if matchesEmissionSelector(function, selector, qualified) {
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
	plan, err := analyzeEmissionTrace(context.Background(), program, entry, profile, storageTransfers, func(from, to *mir.Function, reason string) {
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

func matchesEmissionSelector(function *mir.Function, selector string, qualified bool) bool {
	name := function.Name
	if !qualified {
		if index := strings.LastIndex(name, "::"); index >= 0 {
			name = name[index+len("::"):]
		}
		return name == selector
	}
	prefix := hir.QualifiedModulePath(function.Span.Package, "")
	if name == selector || strings.TrimPrefix(name, prefix) == selector {
		return true
	}
	resource := function.DropGlueFor.Descriptor()
	if resource.Kind != hir.TypeKindNominal {
		return false
	}
	owner, cleanup := strings.CutSuffix(selector, "::cleanup")
	return cleanup && (owner == resource.Name || owner == strings.TrimPrefix(resource.Name, prefix))
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
