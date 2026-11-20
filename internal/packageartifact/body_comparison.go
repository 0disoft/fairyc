package packageartifact

import (
	"crypto/sha256"
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// ExecutableBodyFingerprints compares the stable executable graph for each
// function. Source positions and diagnostic entity spellings are excluded;
// stable call and callback targets remain part of the projection.
func ExecutableBodyFingerprints(artifact Artifact, maxBytes uint64, aliases map[string]string) (map[string][sha256.Size]byte, error) {
	stable, err := DescribeStableArtifact(artifact, maxBytes)
	if err != nil {
		return nil, err
	}
	rewrite := func(value string) string {
		if len(aliases) == 0 || !strings.Contains(value, "_fy_generic_") {
			return value
		}
		return concreteNamePattern.ReplaceAllStringFunc(value, func(name string) string {
			if alias, found := aliases[strings.TrimPrefix(name, "_fy_generic_")]; found {
				return alias
			}
			return name
		})
	}
	targets, err := comparisonTargetIDs(artifact, rewrite, len(aliases) != 0)
	if err != nil {
		return nil, err
	}
	fingerprints := make(map[string][sha256.Size]byte, len(stable.Functions))
	for _, body := range stable.Functions {
		function := body.Function
		if len(aliases) != 0 {
			if err := mapFunctionTypes(&function, func(typ hir.Type) (hir.Type, error) {
				value := typ.String()
				if mapped := rewrite(value); mapped != value {
					return hir.ParseType(mapped), nil
				}
				return typ, nil
			}); err != nil {
				return nil, err
			}
		}
		entities := make(map[string]string)
		normalize := func(value string) string {
			if value == "" {
				return ""
			}
			if normalized, found := entities[value]; found {
				return normalized
			}
			normalized := strconv.Itoa(len(entities) + 1)
			entities[value] = normalized
			return normalized
		}
		for i := range function.Blocks {
			block := &function.Blocks[i]
			block.Entity = normalize(block.Entity)
			block.Terminator.Entity = normalize(block.Terminator.Entity)
			block.Terminator.Span, block.Terminator.UnsafeScope = hir.Span{}, hir.Span{}
			for j := range block.Operations {
				op := &block.Operations[j]
				op.Entity = normalize(op.Entity)
				op.Destination = normalize(op.Destination)
				op.ProjectionEntity = normalize(op.ProjectionEntity)
				op.DropFlagEntity = normalize(op.DropFlagEntity)
				op.Span, op.UnsafeScope = hir.Span{}, hir.Span{}
				if op.Kind == mir.OperationCCallback || op.Kind == mir.OperationClosureCreate {
					if target, found := targets[op.Variant]; found {
						op.Variant = target
					}
				}
			}
		}
		foreign := make(map[uint64]ContextForeignFunction)
		for _, call := range body.Calls {
			if index, ok := strings.CutPrefix(call.Target, bodyForeignCallPrefix); ok {
				// DescribeStableArtifact has already validated this canonical index.
				n, _ := strconv.ParseUint(index, 10, 64)
				foreign[call.Block] = stable.Context.ForeignFunctions[n]
			}
		}
		calls := body.Calls
		for i := range calls {
			if _, found := foreign[calls[i].Block]; found {
				calls[i].Target = bodyForeignCallPrefix
			} else if target, found := targets[strings.TrimPrefix(calls[i].Target, bodyInternalCallPrefix)]; found {
				calls[i].Target = bodyInternalCallPrefix + target
			}
		}
		encoded, err := json.Marshal(struct {
			Entry      mir.BlockID
			Parameters []mir.LocalID
			Blocks     []mir.Block
			Calls      []StableBodyCall
			Foreign    map[uint64]ContextForeignFunction
		}{function.Entry, function.Parameters, function.Blocks, calls, foreign})
		if err != nil {
			return nil, err
		}
		fingerprints[function.Name] = sha256.Sum256(encoded)
	}
	return fingerprints, nil
}

var concreteNamePattern = regexp.MustCompile(`\b_fy_generic_g0_[0-9a-f]{64}\b`)

// Rebind call identities only in the comparison projection. Verified artifacts
// and their persistent cache keys retain the exact compiler-issued identities.
func comparisonTargetIDs(artifact Artifact, rewrite func(string) string, enabled bool) (map[string]string, error) {
	if !enabled {
		return nil, nil
	}
	result := make(map[string]string)
	add := func(id FunctionID, descriptor FunctionDescriptor) error {
		descriptor.Name = rewrite(descriptor.Name)
		descriptor.ResultType = rewrite(descriptor.ResultType)
		descriptor.DropGlueFor = rewrite(descriptor.DropGlueFor)
		descriptor.ParameterTypes = slices.Clone(descriptor.ParameterTypes)
		for i, value := range descriptor.ParameterTypes {
			descriptor.ParameterTypes[i] = rewrite(value)
		}
		normalized, err := ComputeFunctionID(descriptor)
		if err != nil {
			return err
		}
		if id != normalized {
			result[id.String()] = normalized.String()
		}
		return nil
	}
	for _, record := range artifact.Functions {
		if err := add(record.ID, record.Descriptor); err != nil {
			return nil, err
		}
	}
	for _, record := range artifact.Imports {
		if err := add(record.ID, record.Descriptor); err != nil {
			return nil, err
		}
	}
	return result, nil
}
