package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// PreparedProgram owns verified, immutable MIR and its lookup indexes. Each
// execution creates independent runtime state; prepared programs may be reused concurrently.
type PreparedProgram struct {
	byName        map[string]*mir.Function
	bySymbol      map[hir.SymbolID]*mir.Function
	structTypes   map[hir.Type]*mir.StructType
	enumTypes     map[hir.Type]*mir.EnumType
	resourceTypes map[hir.Type]*mir.ResourceType
}

// Prepare takes an independent copy before verification so later caller mutations
// cannot invalidate the verified program. It does not execute any MIR.
func Prepare(program mir.Program) (*PreparedProgram, error) {
	return prepareProgram(mir.Clone(program))
}

// prepareProgram borrows its input only for synchronous single-entry execution.
// Reusable programs must enter through Prepare, which detaches the input first.
func prepareProgram(program mir.Program) (*PreparedProgram, error) {
	if failures := mir.Verify(program); len(failures) != 0 {
		return nil, fmt.Errorf("verify MIR before interpretation: %v", failures)
	}
	prepared := &PreparedProgram{
		byName:        make(map[string]*mir.Function, len(program.Functions)),
		bySymbol:      make(map[hir.SymbolID]*mir.Function, len(program.Functions)),
		structTypes:   make(map[hir.Type]*mir.StructType, len(program.StructTypes)),
		enumTypes:     make(map[hir.Type]*mir.EnumType, len(program.EnumTypes)),
		resourceTypes: make(map[hir.Type]*mir.ResourceType, len(program.ResourceTypes)),
	}
	for i := range program.StructTypes {
		definition := &program.StructTypes[i]
		prepared.structTypes[definition.Type] = definition
	}
	for i := range program.EnumTypes {
		definition := &program.EnumTypes[i]
		prepared.enumTypes[definition.Type] = definition
	}
	for i := range program.ResourceTypes {
		definition := &program.ResourceTypes[i]
		prepared.resourceTypes[definition.Type] = definition
	}
	for i := range program.Functions {
		function := &program.Functions[i]
		if _, duplicate := prepared.byName[function.Name]; duplicate {
			return nil, fmt.Errorf("duplicate MIR function name %q", function.Name)
		}
		if _, duplicate := prepared.bySymbol[function.Symbol]; duplicate {
			return nil, fmt.Errorf("duplicate MIR function symbol %d", function.Symbol)
		}
		prepared.byName[function.Name] = function
		prepared.bySymbol[function.Symbol] = function
	}
	return prepared, nil
}

// ExecuteVoidWithOptions runs one semantic-test entry with fresh budgets, output,
// resource state and authenticated adapters, without repeating MIR verification.
func (prepared *PreparedProgram) ExecuteVoidWithOptions(entry string, options Options) (Outcome, error) {
	if prepared == nil || prepared.byName == nil {
		return Outcome{}, fmt.Errorf("interpreter program is not prepared")
	}
	options, err := options.normalized()
	if err != nil {
		return Outcome{}, err
	}
	if err := options.Context.Err(); err != nil {
		return Outcome{Kind: OutcomeCanceled}, err
	}
	return prepared.executeEntry(entry, hir.TypeVoid, options)
}
