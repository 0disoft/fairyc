package packageartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const (
	StableArtifactSchema  = "fairyc.stable-package-mir/v1"
	StableArtifactVersion = "fairyc-stable-package-mir/3"
)

var (
	ErrStableArtifactTooLarge = errors.New("stable package MIR artifact exceeds the caller byte limit")
	ErrStableArtifactEncoding = errors.New("stable package MIR artifact is not canonically encoded")
)

// StableArtifact is one package-owned, cross-process MIR slice. Context is the closed projection
// required by this package's bodies, not a duplicate of the complete workspace type context.
type StableArtifact struct {
	Package    packagegraph.InstanceID   `json:"package"`
	Coordinate packagegraph.CoordinateID `json:"coordinate"`
	Context    ContextDescriptor         `json:"context"`
	Functions  []StableBodyArtifact      `json:"functions"`
	Imports    []FunctionID              `json:"imports"`
}

type stableArtifactEnvelope struct {
	Schema   string         `json:"schema"`
	Version  string         `json:"version"`
	Artifact StableArtifact `json:"artifact"`
}

// StableFunctionBinding records the fresh process-local symbol assigned to one
// coordinate-owned logical function during stable MIR reassembly.
type StableFunctionBinding struct {
	Package packagegraph.CoordinateID
	Name    string
	Symbol  hir.SymbolID
}

// StableReassembly contains one verified MIR program plus the exact function
// bindings needed to align separately serialized compiler artifacts.
type StableReassembly struct {
	MIR       mir.Program
	Functions []StableFunctionBinding
}

// DescribeStableArtifact projects one verified process-local package artifact into stable bodies,
// explicit FunctionID imports, and the exact stable type context shared by those bodies.
func DescribeStableArtifact(artifact Artifact, maxBytes uint64) (StableArtifact, error) {
	return describeStableArtifact(artifact, maxBytes, false)
}

func describeStableArtifact(artifact Artifact, maxBytes uint64, transfers bool) (StableArtifact, error) {
	verify := Verify
	if transfers {
		verify = VerifyStorageTransfers
	}
	if err := verify(artifact); err != nil {
		return StableArtifact{}, fmt.Errorf("verify process-local artifact before stable projection: %w", err)
	}
	context, foreign, err := projectArtifactContext(artifact)
	if err != nil {
		return StableArtifact{}, err
	}
	result := StableArtifact{Package: artifact.Package, Coordinate: artifact.PackageCoordinate, Context: context}
	for _, record := range artifact.Functions {
		body, err := describeStableBody(record, context, artifact.PackageCoordinates, artifactBindings(artifact), foreign, true, maxBytes)
		if err != nil {
			return StableArtifact{}, fmt.Errorf("describe stable function body %s: %w", record.ID, err)
		}
		result.Functions = append(result.Functions, body)
	}
	result.Imports, err = stableArtifactImports(artifact, context)
	if err != nil {
		return StableArtifact{}, err
	}
	if err := validateStableArtifact(result, transfers); err != nil {
		return StableArtifact{}, err
	}
	return result, nil
}

// ValidateStableArtifact checks package ownership, context identity, sorted bindings, and exact
// resolution of every internal call and resource-drop reference without process-local symbols.
func ValidateStableArtifact(artifact StableArtifact) error {
	return validateStableArtifact(artifact, false)
}

func validateStableArtifact(artifact StableArtifact, transfers bool) error {
	if artifact.Package == (packagegraph.InstanceID{}) {
		return fmt.Errorf("stable package MIR artifact has an empty PackageInstanceId")
	}
	if artifact.Coordinate == (packagegraph.CoordinateID{}) {
		return fmt.Errorf("stable package MIR artifact has an empty PackageCoordinateId")
	}
	if err := ValidateContextDescriptor(artifact.Context); err != nil {
		return err
	}
	contextID, err := ComputeContextID(artifact.Context, ^uint64(0))
	if err != nil {
		return err
	}
	owned := make(map[FunctionID]struct{}, len(artifact.Functions))
	for index, body := range artifact.Functions {
		if index > 0 && bytes.Compare(artifact.Functions[index-1].ID[:], body.ID[:]) >= 0 {
			return fmt.Errorf("stable package MIR functions are not sorted and unique")
		}
		if body.Package != artifact.Coordinate || body.ContextID != contextID {
			return fmt.Errorf("stable package MIR function identity or context is inconsistent")
		}
		if err := ValidateStableBody(body); err != nil {
			return fmt.Errorf("stable function %s: %w", body.ID, err)
		}
		if !transfers {
			for _, block := range body.Function.Blocks {
				for _, operation := range block.Operations {
					if operation.Kind == mir.OperationStorageLayout || operation.Kind == mir.OperationStorageTake || operation.Kind == mir.OperationStoragePut {
						return fmt.Errorf("ordinary stable package MIR does not admit private storage operations")
					}
				}
			}
		}
		owned[body.ID] = struct{}{}
	}
	if len(owned) == 0 {
		return fmt.Errorf("stable package MIR artifact has no owned functions")
	}
	imports := make(map[FunctionID]struct{}, len(artifact.Imports))
	for index, id := range artifact.Imports {
		if id == (FunctionID{}) || index > 0 && bytes.Compare(artifact.Imports[index-1][:], id[:]) >= 0 {
			return fmt.Errorf("stable package MIR imports are not sorted and unique")
		}
		if _, duplicate := owned[id]; duplicate {
			return fmt.Errorf("stable package MIR FunctionID %s is both owned and imported", id)
		}
		imports[id] = struct{}{}
	}

	referencedImports := make(map[FunctionID]struct{})
	recordReference := func(id FunctionID) error {
		if _, local := owned[id]; local {
			return nil
		}
		if _, imported := imports[id]; !imported {
			return fmt.Errorf("stable package MIR reference %s is neither owned nor imported", id)
		}
		referencedImports[id] = struct{}{}
		return nil
	}
	for _, body := range artifact.Functions {
		for _, call := range body.Calls {
			switch {
			case strings.HasPrefix(call.Target, bodyInternalCallPrefix):
				id, err := ParseFunctionID(strings.TrimPrefix(call.Target, bodyInternalCallPrefix))
				if err != nil {
					return err
				}
				if err := recordReference(id); err != nil {
					return err
				}
			case strings.HasPrefix(call.Target, bodyForeignCallPrefix):
				index, err := parseForeignCallTarget(call.Target)
				if err != nil {
					return err
				}
				if index >= uint64(len(artifact.Context.ForeignFunctions)) {
					return fmt.Errorf("stable foreign call index %d is outside the type context", index)
				}
			}
		}
	}
	for _, resource := range artifact.Context.Resources {
		if resource.DropFunctionID != (FunctionID{}) {
			if err := recordReference(resource.DropFunctionID); err != nil {
				return err
			}
		}
	}
	if !reflect.DeepEqual(imports, referencedImports) {
		return fmt.Errorf("stable package MIR imports are not the exact external reference set")
	}
	return nil
}

// EncodeStableArtifact returns canonical exact-version JSON under a mandatory caller limit.
func EncodeStableArtifact(artifact StableArtifact, maxBytes uint64) ([]byte, error) {
	return encodeStableArtifact(artifact, maxBytes, false)
}

func encodeStableArtifact(artifact StableArtifact, maxBytes uint64, transfers bool) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("stable package MIR artifact byte limit must be positive")
	}
	if err := validateStableArtifact(artifact, transfers); err != nil {
		return nil, err
	}
	schema, version := stableArtifactContract(transfers)
	encoded, err := json.Marshal(stableArtifactEnvelope{Schema: schema, Version: version, Artifact: artifact})
	if err != nil {
		return nil, fmt.Errorf("encode stable package MIR artifact: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrStableArtifactTooLarge
	}
	return encoded, nil
}

// DecodeStableArtifact enforces canonical JSON, exact schema/version, strict fields, caller bounds,
// and full stable package validation before returning reusable compiler state.
func DecodeStableArtifact(encoded []byte, maxBytes uint64) (StableArtifact, error) {
	return decodeStableArtifact(encoded, maxBytes, false)
}

func decodeStableArtifact(encoded []byte, maxBytes uint64, transfers bool) (StableArtifact, error) {
	if maxBytes == 0 {
		return StableArtifact{}, fmt.Errorf("stable package MIR artifact byte limit must be positive")
	}
	if len(encoded) == 0 {
		return StableArtifact{}, fmt.Errorf("stable package MIR artifact is empty")
	}
	if uint64(len(encoded)) > maxBytes {
		return StableArtifact{}, ErrStableArtifactTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope stableArtifactEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return StableArtifact{}, fmt.Errorf("decode stable package MIR artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return StableArtifact{}, fmt.Errorf("decode stable package MIR artifact: trailing JSON value")
		}
		return StableArtifact{}, fmt.Errorf("decode stable package MIR artifact trailing data: %w", err)
	}
	schema, version := stableArtifactContract(transfers)
	if envelope.Schema != schema || envelope.Version != version {
		return StableArtifact{}, fmt.Errorf("stable package MIR artifact schema or version is incompatible")
	}
	if err := validateStableArtifact(envelope.Artifact, transfers); err != nil {
		return StableArtifact{}, fmt.Errorf("verify decoded stable package MIR artifact: %w", err)
	}
	canonical, err := encodeStableArtifact(envelope.Artifact, maxBytes, transfers)
	if err != nil {
		return StableArtifact{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return StableArtifact{}, ErrStableArtifactEncoding
	}
	return envelope.Artifact, nil
}

// RebindStableArtifacts reconstructs one closed verified MIR program without any process-local
// package artifact. Package-scoped contexts are conflict-checked and merged before bodies bind.
func RebindStableArtifacts(artifacts []StableArtifact, coordinates map[hir.PackageID]packagegraph.CoordinateID, firstFunctionSymbol, firstForeignSymbol hir.SymbolID, firstLocal mir.LocalID) (mir.Program, error) {
	reassembled, err := RebindStableArtifactsWithBindings(artifacts, coordinates, firstFunctionSymbol, firstForeignSymbol, firstLocal)
	return reassembled.MIR, err
}

// RebindStableArtifactsWithBindings reconstructs one closed verified MIR
// program and returns its deterministic fresh function-symbol assignments.
func RebindStableArtifactsWithBindings(artifacts []StableArtifact, coordinates map[hir.PackageID]packagegraph.CoordinateID, firstFunctionSymbol, firstForeignSymbol hir.SymbolID, firstLocal mir.LocalID) (StableReassembly, error) {
	return rebindStableArtifactsWithBindings(artifacts, coordinates, firstFunctionSymbol, firstForeignSymbol, firstLocal, false)
}

func rebindStableArtifactsWithBindings(artifacts []StableArtifact, coordinates map[hir.PackageID]packagegraph.CoordinateID, firstFunctionSymbol, firstForeignSymbol hir.SymbolID, firstLocal mir.LocalID, transfers bool) (StableReassembly, error) {
	if len(artifacts) == 0 {
		return StableReassembly{}, fmt.Errorf("no stable package MIR artifacts to rebind")
	}
	seenPackages := make(map[packagegraph.InstanceID]struct{}, len(artifacts))
	seenCoordinates := make(map[packagegraph.CoordinateID]struct{}, len(artifacts))
	type ownedBody struct {
		body    StableBodyArtifact
		context ContextDescriptor
	}
	owned := make(map[FunctionID]ownedBody)
	contexts := make([]ContextDescriptor, 0, len(artifacts))
	for index, artifact := range artifacts {
		if err := validateStableArtifact(artifact, transfers); err != nil {
			return StableReassembly{}, fmt.Errorf("stable artifact %d: %w", index, err)
		}
		contexts = append(contexts, artifact.Context)
		if _, duplicate := seenPackages[artifact.Package]; duplicate {
			return StableReassembly{}, fmt.Errorf("stable package MIR artifact %s is duplicated", artifact.Package)
		}
		seenPackages[artifact.Package] = struct{}{}
		if _, duplicate := seenCoordinates[artifact.Coordinate]; duplicate {
			return StableReassembly{}, fmt.Errorf("stable package MIR coordinate %s is duplicated", artifact.Coordinate)
		}
		seenCoordinates[artifact.Coordinate] = struct{}{}
		for _, body := range artifact.Functions {
			if _, duplicate := owned[body.ID]; duplicate {
				return StableReassembly{}, fmt.Errorf("stable owned FunctionID %s is duplicated", body.ID)
			}
			owned[body.ID] = ownedBody{body: body, context: artifact.Context}
		}
	}
	for _, artifact := range artifacts {
		for _, imported := range artifact.Imports {
			if _, found := owned[imported]; !found {
				return StableReassembly{}, fmt.Errorf("stable import FunctionID %s is unresolved", imported)
			}
		}
	}
	ids := make([]FunctionID, 0, len(owned))
	for id := range owned {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(left, right FunctionID) int { return bytes.Compare(left[:], right[:]) })
	functionSymbols := make(map[FunctionID]hir.SymbolID, len(ids))
	for index, id := range ids {
		symbol := firstFunctionSymbol + hir.SymbolID(index)
		if symbol < firstFunctionSymbol || symbol == hir.NoSymbol {
			return StableReassembly{}, fmt.Errorf("stable function symbol namespace overflows")
		}
		functionSymbols[id] = symbol
	}
	mergedContext, err := MergeContextDescriptors(contexts)
	if err != nil {
		return StableReassembly{}, err
	}
	program, err := RebindContextDescriptor(mergedContext, coordinates, functionSymbols, firstForeignSymbol)
	if err != nil {
		return StableReassembly{}, err
	}
	bindings := make([]StableFunctionBinding, 0, len(ids))
	processContext := mir.Clone(program)
	for _, id := range ids {
		item := owned[id]
		bindings = append(bindings, StableFunctionBinding{
			Package: item.body.Package, Name: item.body.Function.Name, Symbol: functionSymbols[id],
		})
		foreignSymbols, foreignErr := foreignSymbolsForContext(processContext, item.context, coordinates)
		if foreignErr != nil {
			return StableReassembly{}, fmt.Errorf("rebind stable function %s foreign context: %w", id, foreignErr)
		}
		function, err := rebindStableBody(item.body, item.context, coordinates, functionSymbols, foreignSymbols, firstLocal, ^uint64(0))
		if err != nil {
			return StableReassembly{}, fmt.Errorf("rebind stable function %s: %w", id, err)
		}
		program.Functions = append(program.Functions, function)
	}
	slices.SortFunc(program.Functions, func(left, right mir.Function) int { return strings.Compare(left.Name, right.Name) })
	verify := mir.Verify
	if transfers {
		verify = mir.VerifyStorageTransfers
	}
	if failures := verify(program); len(failures) != 0 {
		return StableReassembly{}, fmt.Errorf("verify rebound stable package MIR: %w", failures[0])
	}
	return StableReassembly{MIR: program, Functions: bindings}, nil
}
