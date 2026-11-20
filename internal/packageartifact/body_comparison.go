package packageartifact

import (
	"crypto/sha256"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// ExecutableBodyFingerprints compares the stable executable graph for each
// function. Source positions and diagnostic entity spellings are excluded;
// stable call and callback targets remain part of the projection.
func ExecutableBodyFingerprints(artifact Artifact, maxBytes uint64) (map[string][sha256.Size]byte, error) {
	stable, err := DescribeStableArtifact(artifact, maxBytes)
	if err != nil {
		return nil, err
	}
	fingerprints := make(map[string][sha256.Size]byte, len(stable.Functions))
	for _, body := range stable.Functions {
		function := body.Function
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
