package packageartifact

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/mir"
)

func validateStableArtifactFunctions(artifact StableArtifact, contextID ContextID, transfers bool) (map[FunctionID]struct{}, error) {
	owned := make(map[FunctionID]struct{}, len(artifact.Functions))
	for index, body := range artifact.Functions {
		if index > 0 && bytes.Compare(artifact.Functions[index-1].ID[:], body.ID[:]) >= 0 {
			return nil, fmt.Errorf("stable package MIR functions are not sorted and unique")
		}
		if body.Package != artifact.Coordinate || body.ContextID != contextID {
			return nil, fmt.Errorf("stable package MIR function identity or context is inconsistent")
		}
		if err := ValidateStableBody(body); err != nil {
			return nil, fmt.Errorf("stable function %s: %w", body.ID, err)
		}
		if !transfers && containsPrivateStorageOperations(body.Function) {
			return nil, fmt.Errorf("ordinary stable package MIR does not admit private storage operations")
		}
		owned[body.ID] = struct{}{}
	}
	if len(owned) == 0 {
		return nil, fmt.Errorf("stable package MIR artifact has no owned functions")
	}
	return owned, nil
}

func containsPrivateStorageOperations(function mir.Function) bool {
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			switch operation.Kind {
			case mir.OperationStorageLayout, mir.OperationStorageTake, mir.OperationStoragePut, mir.OperationStorageBorrow, mir.OperationStorageBorrowMut:
				return true
			}
		}
	}
	return false
}

func stableArtifactImportUsage(ids []FunctionID, owned map[FunctionID]struct{}) (map[FunctionID]bool, error) {
	imports := make(map[FunctionID]bool, len(ids))
	for index, id := range ids {
		if id == (FunctionID{}) || index > 0 && bytes.Compare(ids[index-1][:], id[:]) >= 0 {
			return nil, fmt.Errorf("stable package MIR imports are not sorted and unique")
		}
		if _, duplicate := owned[id]; duplicate {
			return nil, fmt.Errorf("stable package MIR FunctionID %s is both owned and imported", id)
		}
		imports[id] = false
	}
	return imports, nil
}

// Both sets are local validation indexes, detached from the artifact. Each
// external ID changes from unused to used once, even across repeated calls.
type stableArtifactReferenceCheck struct {
	owned     map[FunctionID]struct{}
	imports   map[FunctionID]bool
	remaining int
}

func (check *stableArtifactReferenceCheck) record(id FunctionID) error {
	if _, local := check.owned[id]; local {
		return nil
	}
	used, imported := check.imports[id]
	if !imported {
		return fmt.Errorf("stable package MIR reference %s is neither owned nor imported", id)
	}
	if !used {
		check.imports[id] = true
		check.remaining--
	}
	return nil
}

func validateStableArtifactReferences(artifact StableArtifact, owned map[FunctionID]struct{}) error {
	imports, err := stableArtifactImportUsage(artifact.Imports, owned)
	if err != nil {
		return err
	}
	check := stableArtifactReferenceCheck{owned: owned, imports: imports, remaining: len(imports)}
	for _, body := range artifact.Functions {
		if err := validateStableFunctionReferences(body, len(artifact.Context.ForeignFunctions), check.record); err != nil {
			return err
		}
	}
	for _, resource := range artifact.Context.Resources {
		if resource.DropFunctionID == (FunctionID{}) {
			continue
		}
		if err := check.record(resource.DropFunctionID); err != nil {
			return err
		}
	}
	if check.remaining != 0 {
		return fmt.Errorf("stable package MIR imports are not the exact external reference set")
	}
	return nil
}

func validateStableFunctionReferences(body StableBodyArtifact, foreignCount int, record func(FunctionID) error) error {
	for _, block := range body.Function.Blocks {
		for _, op := range block.Operations {
			if op.Kind != mir.OperationCCallback && op.Kind != mir.OperationClosureCreate {
				continue
			}
			id, err := ParseFunctionID(op.Variant)
			if err != nil {
				return err
			}
			if err := record(id); err != nil {
				return err
			}
		}
	}
	for _, call := range body.Calls {
		id, internal, err := stableCallReference(call.Target, foreignCount)
		if err != nil {
			return err
		}
		if !internal {
			continue
		}
		if err := record(id); err != nil {
			return err
		}
	}
	return nil
}

func stableCallReference(target string, foreignCount int) (FunctionID, bool, error) {
	if strings.HasPrefix(target, bodyInternalCallPrefix) {
		id, err := ParseFunctionID(strings.TrimPrefix(target, bodyInternalCallPrefix))
		return id, true, err
	}
	if !strings.HasPrefix(target, bodyForeignCallPrefix) {
		return FunctionID{}, false, nil
	}
	index, err := parseForeignCallTarget(target)
	if err != nil {
		return FunctionID{}, false, err
	}
	if index >= uint64(foreignCount) {
		return FunctionID{}, false, fmt.Errorf("stable foreign call index %d is outside the type context", index)
	}
	return FunctionID{}, false, nil
}
