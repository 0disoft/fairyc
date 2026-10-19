package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// The caller authenticates SDK package owners. Verified MIR supplies the exact
// function/type relationship; source spelling alone never enables this model.
func (r *executionState) configureAbstractArenas(options Options) error {
	if len(options.AbstractArenaSDKPackages) == 0 {
		return nil
	}
	r.arenaConstructors = make(map[hir.SymbolID]*mir.ResourceType)
	r.abstractArenas = make(map[*resourceState]bool)
	for _, owner := range options.AbstractArenaSDKPackages {
		module := hir.QualifiedModulePath(owner, "std::alloc")
		create := r.byName[module+"::create"]
		if create == nil {
			continue
		}
		arena, failure, ok := hir.ResultElements(create.ResultType)
		definition := r.resourceTypes[arena]
		errorType := r.structTypes[failure]
		if !ok || len(create.Parameters) != 0 || create.Span.Package != owner ||
			arena.String() != module+"::Arena" || failure.String() != module+"::AllocError" ||
			definition == nil || definition.Span.Package != owner || len(definition.Fields) != 3 ||
			definition.DropKind != hir.ResourceDropNonEmpty || errorType == nil || len(errorType.Fields) != 1 ||
			errorType.Fields[0].Name != "status" || errorType.Fields[0].Type != hir.TypeU32 {
			return fmt.Errorf("authenticated Arena SDK has incompatible constructor or layout")
		}
		fields := definition.Fields
		if fields[0].Name != "head" || fields[0].Type != hir.RawPointer(hir.TypeVoid, true) ||
			fields[1].Name != "generation" || fields[1].Type != hir.TypeUSize ||
			fields[2].Name != "active_region" || fields[2].Type != hir.TypeU8 {
			return fmt.Errorf("authenticated Arena SDK has incompatible fields")
		}
		glue := r.bySymbol[definition.DropSymbol]
		if glue == nil || glue.DropGlueFor != arena || glue.Span.Package != owner {
			return fmt.Errorf("authenticated Arena SDK has incompatible cleanup")
		}
		r.arenaConstructors[create.Symbol] = definition
	}
	return nil
}

func (r *executionState) createAbstractArena(definition *mir.ResourceType) (runtimeValue, error) {
	value, err := resourceValue(definition, []runtimeValue{
		{kind: valueNull, typ: definition.Fields[0].Type},
		{kind: valueUSize, typ: hir.TypeUSize, scalar: 1},
		{kind: valueU8, typ: hir.TypeU8},
	})
	if err != nil {
		return runtimeValue{}, err
	}
	r.abstractArenas[value.resource()] = true
	return resultValue("ok", value)
}
