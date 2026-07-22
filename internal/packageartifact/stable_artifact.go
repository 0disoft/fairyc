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
	StableArtifactVersion = "fairyc-stable-package-mir/1"
)

var (
	ErrStableArtifactTooLarge = errors.New("stable package MIR artifact exceeds the caller byte limit")
	ErrStableArtifactEncoding = errors.New("stable package MIR artifact is not canonically encoded")
)

// StableArtifact is one package-owned, cross-process MIR slice. It carries the complete stable
// workspace type context until package-scoped context projection is separately proven.
type StableArtifact struct {
	Package   packagegraph.InstanceID `json:"package"`
	Context   ContextDescriptor       `json:"context"`
	Functions []StableBodyArtifact    `json:"functions"`
	Imports   []FunctionID            `json:"imports"`
}

type stableArtifactEnvelope struct {
	Schema   string         `json:"schema"`
	Version  string         `json:"version"`
	Artifact StableArtifact `json:"artifact"`
}

// DescribeStableArtifact projects one verified process-local package artifact into stable bodies,
// explicit FunctionID imports, and the exact stable type context shared by those bodies.
func DescribeStableArtifact(artifact Artifact, maxBytes uint64) (StableArtifact, error) {
	if err := Verify(artifact); err != nil {
		return StableArtifact{}, fmt.Errorf("verify process-local artifact before stable projection: %w", err)
	}
	result := StableArtifact{Package: artifact.Package, Context: cloneContextDescriptor(artifact.StableContext)}
	for _, record := range artifact.Functions {
		body, err := DescribeStableBody(record, artifact, maxBytes)
		if err != nil {
			return StableArtifact{}, fmt.Errorf("describe stable function body %s: %w", record.ID, err)
		}
		result.Functions = append(result.Functions, body)
	}
	for _, imported := range artifact.Imports {
		result.Imports = append(result.Imports, imported.ID)
	}
	if err := ValidateStableArtifact(result); err != nil {
		return StableArtifact{}, err
	}
	return result, nil
}

// ValidateStableArtifact checks package ownership, context identity, sorted bindings, and exact
// resolution of every internal call and resource-drop reference without process-local symbols.
func ValidateStableArtifact(artifact StableArtifact) error {
	if artifact.Package == (packagegraph.InstanceID{}) {
		return fmt.Errorf("stable package MIR artifact has an empty PackageInstanceId")
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
		if body.Package != artifact.Package || body.ContextID != contextID {
			return fmt.Errorf("stable package MIR function identity or context is inconsistent")
		}
		if err := ValidateStableBody(body); err != nil {
			return fmt.Errorf("stable function %s: %w", body.ID, err)
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
	if maxBytes == 0 {
		return nil, fmt.Errorf("stable package MIR artifact byte limit must be positive")
	}
	if err := ValidateStableArtifact(artifact); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(stableArtifactEnvelope{Schema: StableArtifactSchema, Version: StableArtifactVersion, Artifact: artifact})
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
	if envelope.Schema != StableArtifactSchema || envelope.Version != StableArtifactVersion {
		return StableArtifact{}, fmt.Errorf("stable package MIR artifact schema or version is incompatible")
	}
	if err := ValidateStableArtifact(envelope.Artifact); err != nil {
		return StableArtifact{}, fmt.Errorf("verify decoded stable package MIR artifact: %w", err)
	}
	canonical, err := EncodeStableArtifact(envelope.Artifact, maxBytes)
	if err != nil {
		return StableArtifact{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return StableArtifact{}, ErrStableArtifactEncoding
	}
	return envelope.Artifact, nil
}

// RebindStableArtifacts reconstructs one closed verified MIR program without any process-local
// package artifact. All stable packages must share one context and resolve every import exactly.
func RebindStableArtifacts(artifacts []StableArtifact, identities map[hir.PackageID]packagegraph.InstanceID, firstFunctionSymbol, firstForeignSymbol hir.SymbolID, firstLocal mir.LocalID) (mir.Program, error) {
	if len(artifacts) == 0 {
		return mir.Program{}, fmt.Errorf("no stable package MIR artifacts to rebind")
	}
	seenPackages := make(map[packagegraph.InstanceID]struct{}, len(artifacts))
	owned := make(map[FunctionID]StableBodyArtifact)
	baseContext := artifacts[0].Context
	for index, artifact := range artifacts {
		if err := ValidateStableArtifact(artifact); err != nil {
			return mir.Program{}, fmt.Errorf("stable artifact %d: %w", index, err)
		}
		if !reflect.DeepEqual(artifact.Context, baseContext) {
			return mir.Program{}, fmt.Errorf("stable package MIR artifacts do not share one type context")
		}
		if _, duplicate := seenPackages[artifact.Package]; duplicate {
			return mir.Program{}, fmt.Errorf("stable package MIR artifact %s is duplicated", artifact.Package)
		}
		seenPackages[artifact.Package] = struct{}{}
		for _, body := range artifact.Functions {
			if _, duplicate := owned[body.ID]; duplicate {
				return mir.Program{}, fmt.Errorf("stable owned FunctionID %s is duplicated", body.ID)
			}
			owned[body.ID] = body
		}
	}
	for _, artifact := range artifacts {
		for _, imported := range artifact.Imports {
			if _, found := owned[imported]; !found {
				return mir.Program{}, fmt.Errorf("stable import FunctionID %s is unresolved", imported)
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
			return mir.Program{}, fmt.Errorf("stable function symbol namespace overflows")
		}
		functionSymbols[id] = symbol
	}
	program, err := RebindContextDescriptor(baseContext, identities, functionSymbols, firstForeignSymbol)
	if err != nil {
		return mir.Program{}, err
	}
	processContext := mir.Clone(program)
	for _, id := range ids {
		function, err := RebindStableBody(owned[id], baseContext, processContext, identities, functionSymbols, firstLocal, ^uint64(0))
		if err != nil {
			return mir.Program{}, fmt.Errorf("rebind stable function %s: %w", id, err)
		}
		program.Functions = append(program.Functions, function)
	}
	slices.SortFunc(program.Functions, func(left, right mir.Function) int { return strings.Compare(left.Name, right.Name) })
	if failures := mir.Verify(program); len(failures) != 0 {
		return mir.Program{}, fmt.Errorf("verify rebound stable package MIR: %w", failures[0])
	}
	return program, nil
}
