package packageartifact

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// projectArtifactContext keeps only declarations required by one package's stable bodies. The
// projection is closed over aggregate fields and resource drop links, so unrelated workspace
// declarations cannot change the package artifact bytes.
func projectArtifactContext(artifact Artifact) (ContextDescriptor, map[hir.SymbolID]uint64, error) {
	usedTypes := make(map[string]struct{})
	collectType := func(value hir.Type) (hir.Type, error) {
		stable, err := stableType(value, artifact.PackageIdentities)
		if err != nil {
			return hir.TypeInvalid, err
		}
		collectStableNominalTypes(stable, usedTypes)
		return value, nil
	}
	usedForeign := make(map[hir.SymbolID]struct{})
	for _, record := range artifact.Functions {
		function := cloneFunction(record.Body)
		if err := mapFunctionTypes(&function, collectType); err != nil {
			return ContextDescriptor{}, nil, fmt.Errorf("collect stable function types for %q: %w", record.Body.Name, err)
		}
		for _, block := range record.Body.Blocks {
			terminator := block.Terminator
			if terminator.Kind == mir.TerminatorCall && terminator.ForeignABI != "" {
				usedForeign[terminator.Callee] = struct{}{}
			}
		}
	}

	foreignBySymbol := make(map[hir.SymbolID]ContextForeignFunction, len(artifact.Context.ForeignFunctions))
	for _, foreign := range artifact.Context.ForeignFunctions {
		described, err := DescribeContext(mir.Program{ForeignFunctions: []mir.ForeignFunction{foreign}}, artifact.PackageIdentities, nil)
		if err != nil {
			return ContextDescriptor{}, nil, fmt.Errorf("describe foreign function %q for package context: %w", foreign.Name, err)
		}
		if len(described.ForeignFunctions) != 1 {
			return ContextDescriptor{}, nil, fmt.Errorf("foreign function %q did not produce one stable declaration", foreign.Name)
		}
		foreignBySymbol[foreign.Symbol] = described.ForeignFunctions[0]
	}

	result := ContextDescriptor{}
	for symbol := range usedForeign {
		foreign, found := foreignBySymbol[symbol]
		if !found {
			return ContextDescriptor{}, nil, fmt.Errorf("foreign call symbol %d has no stable declaration", symbol)
		}
		result.ForeignFunctions = append(result.ForeignFunctions, foreign)
		for _, parameterType := range foreign.ParameterTypes {
			collectStableNominalTypes(parameterType, usedTypes)
		}
		collectStableNominalTypes(foreign.ResultType, usedTypes)
	}

	structs := make(map[string]ContextStruct, len(artifact.StableContext.Structs))
	for _, item := range artifact.StableContext.Structs {
		structs[item.Type] = item
	}
	enums := make(map[string]ContextEnum, len(artifact.StableContext.Enums))
	for _, item := range artifact.StableContext.Enums {
		enums[item.Type] = item
	}
	resources := make(map[string]ContextResource, len(artifact.StableContext.Resources))
	for _, item := range artifact.StableContext.Resources {
		resources[item.Type] = item
	}

	visited := make(map[string]struct{})
	for {
		pending := make([]string, 0)
		for name := range usedTypes {
			if _, done := visited[name]; !done {
				pending = append(pending, name)
			}
		}
		if len(pending) == 0 {
			break
		}
		slices.Sort(pending)
		for _, name := range pending {
			visited[name] = struct{}{}
			switch {
			case structs[name].Type != "":
				item := structs[name]
				result.Structs = append(result.Structs, item)
				for _, field := range item.Fields {
					collectStableNominalTypes(field.Type, usedTypes)
				}
			case enums[name].Type != "":
				item := enums[name]
				result.Enums = append(result.Enums, item)
				for _, variant := range item.Variants {
					for _, field := range variant.Fields {
						collectStableNominalTypes(field.Type, usedTypes)
					}
				}
			case resources[name].Type != "":
				item := resources[name]
				result.Resources = append(result.Resources, item)
				for _, field := range item.Fields {
					collectStableNominalTypes(field.Type, usedTypes)
				}
			}
		}
	}

	sortContextDescriptor(&result)
	if err := ValidateContextDescriptor(result); err != nil {
		return ContextDescriptor{}, nil, fmt.Errorf("validate projected package context: %w", err)
	}
	foreignIndices, err := contextForeignIndicesForSymbols(artifact.Context, result, artifact.PackageIdentities, usedForeign)
	if err != nil {
		return ContextDescriptor{}, nil, err
	}
	return result, foreignIndices, nil
}

func collectStableNominalTypes(value string, result map[string]struct{}) {
	if value == "" || builtinType(hir.Type(value)) {
		return
	}
	for _, prefix := range []string{"ref mut ", "ref ", "ptr mut ", "ptr "} {
		if strings.HasPrefix(value, prefix) {
			collectStableNominalTypes(strings.TrimPrefix(value, prefix), result)
			return
		}
	}
	if element, ok := hir.OptionElement(hir.Type(value)); ok {
		collectStableNominalTypes(string(element), result)
		return
	}
	if success, failure, ok := hir.ResultElements(hir.Type(value)); ok {
		collectStableNominalTypes(string(success), result)
		collectStableNominalTypes(string(failure), result)
		return
	}
	if strings.HasSuffix(value, "]") {
		if separator := strings.LastIndexByte(value, '['); separator > 0 {
			collectStableNominalTypes(value[:separator], result)
			return
		}
	}
	if validStableNominalType(value) {
		result[value] = struct{}{}
	}
}

func contextForeignIndicesForSymbols(processContext mir.Program, stable ContextDescriptor, identities map[hir.PackageID]packagegraph.InstanceID, required map[hir.SymbolID]struct{}) (map[hir.SymbolID]uint64, error) {
	result := make(map[hir.SymbolID]uint64, len(required))
	for _, foreign := range processContext.ForeignFunctions {
		if _, needed := required[foreign.Symbol]; !needed {
			continue
		}
		described, err := DescribeContext(mir.Program{ForeignFunctions: []mir.ForeignFunction{foreign}}, identities, nil)
		if err != nil {
			return nil, fmt.Errorf("describe foreign function %q for stable body: %w", foreign.Name, err)
		}
		index := slices.IndexFunc(stable.ForeignFunctions, func(candidate ContextForeignFunction) bool {
			return reflect.DeepEqual(candidate, described.ForeignFunctions[0])
		})
		if index < 0 {
			return nil, fmt.Errorf("foreign function %q is absent from the projected package context", foreign.Name)
		}
		result[foreign.Symbol] = uint64(index)
	}
	if len(result) != len(required) {
		return nil, fmt.Errorf("projected package context did not resolve every foreign call")
	}
	return result, nil
}

// MergeContextDescriptors builds one deterministic whole-program context from independently
// cached package projections. Repeated declarations must be byte-for-byte semantic equals.
func MergeContextDescriptors(descriptors []ContextDescriptor) (ContextDescriptor, error) {
	result := ContextDescriptor{}
	typeKinds := make(map[string]string)
	structs := make(map[string]ContextStruct)
	enums := make(map[string]ContextEnum)
	resources := make(map[string]ContextResource)
	foreign := make(map[string]ContextForeignFunction)

	recordType := func(name, kind string) error {
		if previous, found := typeKinds[name]; found && previous != kind {
			return fmt.Errorf("stable type %q conflicts between %s and %s package contexts", name, previous, kind)
		}
		typeKinds[name] = kind
		return nil
	}
	for index, descriptor := range descriptors {
		if err := ValidateContextDescriptor(descriptor); err != nil {
			return ContextDescriptor{}, fmt.Errorf("package context %d: %w", index, err)
		}
		for _, item := range descriptor.Structs {
			if err := recordType(item.Type, "struct"); err != nil {
				return ContextDescriptor{}, err
			}
			if previous, found := structs[item.Type]; found && !reflect.DeepEqual(previous, item) {
				return ContextDescriptor{}, fmt.Errorf("stable struct %q has conflicting package-context definitions", item.Type)
			}
			structs[item.Type] = item
		}
		for _, item := range descriptor.Enums {
			if err := recordType(item.Type, "enum"); err != nil {
				return ContextDescriptor{}, err
			}
			if previous, found := enums[item.Type]; found && !reflect.DeepEqual(previous, item) {
				return ContextDescriptor{}, fmt.Errorf("stable enum %q has conflicting package-context definitions", item.Type)
			}
			enums[item.Type] = item
		}
		for _, item := range descriptor.Resources {
			if err := recordType(item.Type, "resource"); err != nil {
				return ContextDescriptor{}, err
			}
			if previous, found := resources[item.Type]; found && !reflect.DeepEqual(previous, item) {
				return ContextDescriptor{}, fmt.Errorf("stable resource %q has conflicting package-context definitions", item.Type)
			}
			resources[item.Type] = item
		}
		for _, item := range descriptor.ForeignFunctions {
			key := item.Package.String() + "\x00" + item.Name
			if previous, found := foreign[key]; found && !reflect.DeepEqual(previous, item) {
				return ContextDescriptor{}, fmt.Errorf("stable foreign function %q has conflicting package-context definitions", item.Name)
			}
			foreign[key] = item
		}
	}
	for _, item := range structs {
		result.Structs = append(result.Structs, item)
	}
	for _, item := range enums {
		result.Enums = append(result.Enums, item)
	}
	for _, item := range resources {
		result.Resources = append(result.Resources, item)
	}
	for _, item := range foreign {
		result.ForeignFunctions = append(result.ForeignFunctions, item)
	}
	sortContextDescriptor(&result)
	if err := ValidateContextDescriptor(result); err != nil {
		return ContextDescriptor{}, fmt.Errorf("validate merged package contexts: %w", err)
	}
	return result, nil
}

func stableArtifactImports(artifact Artifact, context ContextDescriptor) ([]FunctionID, error) {
	owned := make(map[FunctionID]struct{}, len(artifact.Functions))
	for _, record := range artifact.Functions {
		owned[record.ID] = struct{}{}
	}
	bindings := artifactBindings(artifact)
	required := make(map[FunctionID]struct{})
	for _, record := range artifact.Functions {
		for _, block := range record.Body.Blocks {
			terminator := block.Terminator
			if terminator.Kind != mir.TerminatorCall || terminator.ForeignABI != "" {
				continue
			}
			id, found := bindings[terminator.Callee]
			if !found {
				return nil, fmt.Errorf("function %q call symbol %d has no artifact binding", record.Body.Name, terminator.Callee)
			}
			if _, local := owned[id]; !local {
				required[id] = struct{}{}
			}
		}
	}
	for _, resource := range context.Resources {
		if resource.DropFunctionID == (FunctionID{}) {
			continue
		}
		if _, local := owned[resource.DropFunctionID]; !local {
			required[resource.DropFunctionID] = struct{}{}
		}
	}
	result := make([]FunctionID, 0, len(required))
	for id := range required {
		result = append(result, id)
	}
	slices.SortFunc(result, func(left, right FunctionID) int { return bytes.Compare(left[:], right[:]) })
	return result, nil
}

func foreignSymbolsForContext(processContext mir.Program, local ContextDescriptor, identities map[hir.PackageID]packagegraph.InstanceID) (map[uint64]hir.SymbolID, error) {
	type binding struct {
		declaration ContextForeignFunction
		symbol      hir.SymbolID
	}
	available := make(map[string]binding, len(processContext.ForeignFunctions))
	for _, foreign := range processContext.ForeignFunctions {
		described, err := DescribeContext(mir.Program{ForeignFunctions: []mir.ForeignFunction{foreign}}, identities, nil)
		if err != nil {
			return nil, err
		}
		declaration := described.ForeignFunctions[0]
		key := declaration.Package.String() + "\x00" + declaration.Name
		if previous, duplicate := available[key]; duplicate && (!reflect.DeepEqual(previous.declaration, declaration) || previous.symbol != foreign.Symbol) {
			return nil, fmt.Errorf("merged context has an ambiguous foreign function %q", declaration.Name)
		}
		available[key] = binding{declaration: declaration, symbol: foreign.Symbol}
	}
	result := make(map[uint64]hir.SymbolID, len(local.ForeignFunctions))
	for index, declaration := range local.ForeignFunctions {
		key := declaration.Package.String() + "\x00" + declaration.Name
		matched, found := available[key]
		if !found || !reflect.DeepEqual(matched.declaration, declaration) {
			return nil, fmt.Errorf("local package context foreign function %q is absent or conflicts in merged context", declaration.Name)
		}
		result[uint64(index)] = matched.symbol
	}
	return result, nil
}
