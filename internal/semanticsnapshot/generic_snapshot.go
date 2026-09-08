package semanticsnapshot

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
)

// BuildCompiler preserves original generic declarations and concrete executable
// summaries separately. The legacy Build path remains unchanged for ordinary code.
func BuildCompiler(result compiler.Result, packages []PackageInput) (Snapshot, error) {
	if !result.Verified {
		return Snapshot{}, fmt.Errorf("compiler snapshot requires a verified result")
	}
	originals := result.GenericProvenance.Declarations()
	if len(originals) == 0 {
		if len(result.GenericProvenance.Specializations()) != 0 {
			return Snapshot{}, fmt.Errorf("specializations have no original declarations")
		}
		return Build(result.Semantic, result.MIR, packages)
	}
	if !result.Semantic.RequiresGenericProvenance {
		return Snapshot{}, fmt.Errorf("generic provenance contradicts ordinary semantic source metadata")
	}
	if err := validateSnapshotInput(result.Semantic, result.MIR); err != nil {
		return Snapshot{}, err
	}
	metadata, stablePackages, files, err := normalizePackages(packages)
	if err != nil {
		return Snapshot{}, err
	}
	modules, concrete, symbols, foreign, private, bounds, err := buildDeclarations(result.Semantic.HIR, metadata)
	if err != nil {
		return Snapshot{}, err
	}
	if err := validatePackageCoverage(stablePackages, files, modules); err != nil {
		return Snapshot{}, err
	}
	generic, err := GenericDeclarations(result.GenericProvenance, packages)
	if err != nil {
		return Snapshot{}, err
	}
	declarations, implementations, err := joinGenericImplementations(result.GenericProvenance, generic, concrete, metadata)
	if err != nil {
		return Snapshot{}, err
	}
	cleanup, ordinaryBorrows, cleanupResolved, cleanupExternal, err := buildGenericCleanupSummaries(result, implementations, metadata, symbols, foreign, private)
	if err != nil {
		return Snapshot{}, err
	}
	functions, resolved, external, err := buildFunctionSummaries(result.MIR, ordinaryBorrows, metadata, symbols, foreign, private, bounds)
	if err != nil {
		return Snapshot{}, err
	}
	var ordinary []FunctionSummary
	var specializations []SpecializationSummary
	seen := map[DeclarationKey]bool{}
	remapCalls := func(function *FunctionSummary) {
		for i := range function.Calls {
			if target, ok := implementations[function.Calls[i].Target]; ok {
				function.Calls[i].Target = target.Declaration
				function.Calls[i].Specialization = target.ID
			}
		}
		slices.SortFunc(function.Calls, compareCall)
	}
	for key, implementation := range implementations {
		if implementation.Aggregate == nil {
			continue
		}
		seen[key] = true
		if summary, found := cleanup[key]; found {
			remapCalls(&summary)
			implementation.Cleanup = &summary
		}
		specializations = append(specializations, implementation)
	}
	for _, function := range functions {
		remapCalls(&function)
		if implementation, ok := implementations[function.Function]; ok {
			seen[function.Function] = true
			function.Function = implementation.Declaration
			implementation.Function = function
			specializations = append(specializations, implementation)
		} else {
			ordinary = append(ordinary, function)
		}
	}
	if len(seen) != len(implementations) {
		return Snapshot{}, fmt.Errorf("generic implementation has no verified executable summary")
	}
	slices.SortFunc(specializations, func(a, b SpecializationSummary) int { return cmp.Compare(a.ID, b.ID) })
	// Documentation belongs to original declarations, never to each generated
	// clone. Filter a detached HIR view used only by the attachment extractor.
	docHIR := result.Semantic.HIR
	docHIR.Modules = slices.Clone(docHIR.Modules)
	for i := range docHIR.Modules {
		module := &docHIR.Modules[i]
		module.Functions = slices.DeleteFunc(slices.Clone(module.Functions), func(function hir.Function) bool {
			_, generated := implementations[symbols[function.Symbol]]
			return generated
		})
		module.Structs = slices.DeleteFunc(slices.Clone(module.Structs), func(value hir.Struct) bool {
			_, found := implementations[DeclarationKey{PackageInstance: metadata[module.Package].instance, Module: module.Path, Kind: DeclarationStruct, Name: value.Name}]
			return found
		})
		module.Resources = slices.DeleteFunc(slices.Clone(module.Resources), func(value hir.Resource) bool {
			_, found := implementations[DeclarationKey{PackageInstance: metadata[module.Package].instance, Module: module.Path, Kind: DeclarationResource, Name: value.Name}]
			return found
		})
	}
	documentation, err := buildDocumentationSpans(docHIR, metadata, declarations)
	if err != nil {
		return Snapshot{}, err
	}
	for _, original := range originals {
		if original.DocumentationSpan == (hir.Span{}) {
			continue
		}
		declarationSpan, err := normalizeSpan(original.Span, metadata)
		if err != nil {
			return Snapshot{}, err
		}
		docSpan, err := normalizeSpan(original.DocumentationSpan, metadata)
		if err != nil || docSpan.Source != declarationSpan.Source || docSpan.PackageInstance != declarationSpan.PackageInstance ||
			docSpan.EndByte <= docSpan.StartByte || docSpan.EndByte > declarationSpan.StartByte {
			return Snapshot{}, fmt.Errorf("generic documentation has no exact preceding source span")
		}
		documentation = append(documentation, DeclarationDocumentation{
			Declaration:     DeclarationKey{PackageInstance: declarationSpan.PackageInstance, Module: original.Module, Kind: DeclarationKind(original.Kind), Name: original.Name},
			DeclarationSpan: declarationSpan, Documentation: docSpan,
		})
	}
	slices.SortFunc(documentation, func(a, b DeclarationDocumentation) int { return compareDeclarationKey(a.Declaration, b.Declaration) })
	completeness := buildCompleteness(resolved+cleanupResolved, external+cleanupExternal)
	completeness = append(completeness,
		Completeness{Relation: "generic-specializations", Status: CompletenessExact, Resolved: len(specializations)},
		Completeness{Relation: "generic-template-bodies", Status: CompletenessUnavailable, Unknown: len(generic), Reasons: []string{"bodies-checked-only-after-specialization"}},
	)
	slices.SortFunc(completeness, func(a, b Completeness) int { return cmp.Compare(a.Relation, b.Relation) })
	return Snapshot{data: snapshotData{packages: stablePackages, files: files, modules: modules, declarations: declarations,
		documentation: documentation, functions: ordinary, specializations: specializations, completeness: completeness}}, nil
}
