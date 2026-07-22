package packageartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const (
	BodySchema  = "fairyc.package-function-body/v0"
	BodyVersion = "fairyc-package-function-body/0"
)

var (
	ErrBodyArtifactTooLarge = errors.New("stable function-body artifact exceeds the caller byte limit")
	ErrBodyArtifactEncoding = errors.New("stable function-body artifact is not canonically encoded")
)

// StableBodyArtifact is the first cross-process function-body format. Version 0 intentionally
// admits only call-free bodies so type, span, entity, local, block, and context rebinding can be
// verified independently before stable internal and foreign call references are added.
type StableBodyArtifact struct {
	ID         FunctionID              `json:"id"`
	Package    packagegraph.InstanceID `json:"package"`
	ContextID  ContextID               `json:"contextId"`
	LocalCount uint64                  `json:"localCount"`
	BlockCount uint64                  `json:"blockCount"`
	Function   mir.Function            `json:"function"`
}

type bodyEnvelope struct {
	Schema  string             `json:"schema"`
	Version string             `json:"version"`
	Body    StableBodyArtifact `json:"body"`
}

// DescribeStableLeafBody removes build-local numbering from one independently verified function
// record and binds the result to the exact stable type context. Calls fail closed in v0.
func DescribeStableLeafBody(record FunctionRecord, context ContextDescriptor, identities map[hir.PackageID]packagegraph.InstanceID, maxBytes uint64) (StableBodyArtifact, error) {
	if err := verifyBinding(record.ID, record.Descriptor, record.Body, identities); err != nil {
		return StableBodyArtifact{}, fmt.Errorf("verify function binding before stable body projection: %w", err)
	}
	contextID, err := ComputeContextID(context, maxBytes)
	if err != nil {
		return StableBodyArtifact{}, fmt.Errorf("compute stable body context id: %w", err)
	}
	function, shape, err := canonicalizeFunctionGraphIDs(record.Body)
	if err != nil {
		return StableBodyArtifact{}, err
	}
	for _, block := range function.Blocks {
		if block.Terminator.Kind == mir.TerminatorCall {
			return StableBodyArtifact{}, fmt.Errorf("stable function-body v0 does not admit calls")
		}
	}
	owner := record.Body.Span.Package
	function.Symbol = hir.NoSymbol
	function.Name = record.Descriptor.Name
	if err := mapFunctionTypes(&function, func(value hir.Type) (hir.Type, error) {
		stable, stableErr := stableType(value, identities)
		return hir.Type(stable), stableErr
	}); err != nil {
		return StableBodyArtifact{}, fmt.Errorf("stabilize function body types: %w", err)
	}
	if err := mapFunctionSpans(&function, func(span hir.Span) (hir.Span, error) {
		if emptySpan(span) {
			return hir.Span{}, nil
		}
		if span.Package != owner || span.Source == "" || span.StartByte < 0 || span.EndByte < span.StartByte {
			return hir.Span{}, fmt.Errorf("span %+v does not belong to function package %d", span, owner)
		}
		span.Package = hir.DefaultPackage
		return span, nil
	}); err != nil {
		return StableBodyArtifact{}, fmt.Errorf("stabilize function body spans: %w", err)
	}
	if err := mapFunctionEntities(&function, func(value string) (string, error) {
		return stabilizeSourceEntity(value, owner)
	}); err != nil {
		return StableBodyArtifact{}, fmt.Errorf("stabilize function body entities: %w", err)
	}
	body := StableBodyArtifact{
		ID: record.ID, Package: record.Descriptor.Package, ContextID: contextID,
		LocalCount: uint64(shape.LocalCount), BlockCount: uint64(shape.BlockCount), Function: function,
	}
	if err := ValidateStableBody(body); err != nil {
		return StableBodyArtifact{}, err
	}
	return body, nil
}

// RebindStableLeafBody restores one decoded v0 body into fresh package, function-symbol, and local
// namespaces. The returned function must still enter mir.Verify in its caller-owned closed world.
func RebindStableLeafBody(body StableBodyArtifact, context ContextDescriptor, identities map[hir.PackageID]packagegraph.InstanceID, functionSymbol hir.SymbolID, firstLocal mir.LocalID, maxBytes uint64) (mir.Function, error) {
	if err := ValidateStableBody(body); err != nil {
		return mir.Function{}, err
	}
	contextID, err := ComputeContextID(context, maxBytes)
	if err != nil {
		return mir.Function{}, fmt.Errorf("compute rebound body context id: %w", err)
	}
	if contextID != body.ContextID {
		return mir.Function{}, fmt.Errorf("stable function body context does not match the requested type context")
	}
	if functionSymbol == hir.NoSymbol {
		return mir.Function{}, fmt.Errorf("stable function body requires a fresh function symbol")
	}
	owner, err := packageForInstance(body.Package, identities)
	if err != nil {
		return mir.Function{}, err
	}
	shape, err := bodyShape(body)
	if err != nil {
		return mir.Function{}, err
	}
	function, err := rebindFunctionGraphIDs(body.Function, shape, firstLocal)
	if err != nil {
		return mir.Function{}, err
	}
	function.Symbol = functionSymbol
	function.Name = hir.QualifiedModulePath(owner, body.Function.Name)
	if err := mapFunctionTypes(&function, func(value hir.Type) (hir.Type, error) {
		return rebindStableType(string(value), identities)
	}); err != nil {
		return mir.Function{}, fmt.Errorf("rebind function body types: %w", err)
	}
	if err := mapFunctionSpans(&function, func(span hir.Span) (hir.Span, error) {
		if emptySpan(span) {
			return hir.Span{}, nil
		}
		span.Package = owner
		return span, nil
	}); err != nil {
		return mir.Function{}, fmt.Errorf("rebind function body spans: %w", err)
	}
	if err := mapFunctionEntities(&function, func(value string) (string, error) {
		return rebindSourceEntity(value, owner)
	}); err != nil {
		return mir.Function{}, fmt.Errorf("rebind function body entities: %w", err)
	}
	descriptor, err := DescribeFunction(function, identities)
	if err != nil {
		return mir.Function{}, err
	}
	computed, err := ComputeFunctionID(descriptor)
	if err != nil {
		return mir.Function{}, err
	}
	if computed != body.ID {
		return mir.Function{}, fmt.Errorf("rebound function identity does not match stable body id")
	}
	return function, nil
}

// EncodeStableBody returns canonical exact-version JSON under a mandatory caller limit.
func EncodeStableBody(body StableBodyArtifact, maxBytes uint64) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("stable function-body artifact byte limit must be positive")
	}
	if err := ValidateStableBody(body); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(bodyEnvelope{Schema: BodySchema, Version: BodyVersion, Body: body})
	if err != nil {
		return nil, fmt.Errorf("encode stable function-body artifact: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrBodyArtifactTooLarge
	}
	return encoded, nil
}

// DecodeStableBody enforces one canonical JSON value, an exact schema/version, strict fields, and
// complete independent body validation before returning the artifact.
func DecodeStableBody(encoded []byte, maxBytes uint64) (StableBodyArtifact, error) {
	if maxBytes == 0 {
		return StableBodyArtifact{}, fmt.Errorf("stable function-body artifact byte limit must be positive")
	}
	if len(encoded) == 0 {
		return StableBodyArtifact{}, fmt.Errorf("stable function-body artifact is empty")
	}
	if uint64(len(encoded)) > maxBytes {
		return StableBodyArtifact{}, ErrBodyArtifactTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope bodyEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return StableBodyArtifact{}, fmt.Errorf("decode stable function-body artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return StableBodyArtifact{}, fmt.Errorf("decode stable function-body artifact: trailing JSON value")
		}
		return StableBodyArtifact{}, fmt.Errorf("decode stable function-body artifact trailing data: %w", err)
	}
	if envelope.Schema != BodySchema || envelope.Version != BodyVersion {
		return StableBodyArtifact{}, fmt.Errorf("stable function-body schema or version is incompatible")
	}
	if err := ValidateStableBody(envelope.Body); err != nil {
		return StableBodyArtifact{}, fmt.Errorf("verify decoded stable function-body artifact: %w", err)
	}
	canonical, err := EncodeStableBody(envelope.Body, maxBytes)
	if err != nil {
		return StableBodyArtifact{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return StableBodyArtifact{}, ErrBodyArtifactEncoding
	}
	return envelope.Body, nil
}

// ValidateStableBody checks every invariant available without a process-local type context.
func ValidateStableBody(body StableBodyArtifact) error {
	if body.ID == (FunctionID{}) || body.Package == (packagegraph.InstanceID{}) || body.ContextID == (ContextID{}) {
		return fmt.Errorf("stable function-body identity is incomplete")
	}
	if body.Function.Symbol != hir.NoSymbol || !validLogicalFunctionName(body.Function.Name) {
		return fmt.Errorf("stable function body retains a build-local function identity")
	}
	shape, err := bodyShape(body)
	if err != nil {
		return err
	}
	canonical, canonicalShape, err := canonicalizeFunctionGraphIDs(body.Function)
	if err != nil {
		return err
	}
	if canonicalShape != shape || !reflect.DeepEqual(canonical, body.Function) {
		return fmt.Errorf("stable function body graph is not canonical")
	}
	for _, block := range body.Function.Blocks {
		if block.Terminator.Kind == mir.TerminatorCall {
			return fmt.Errorf("stable function-body v0 does not admit calls")
		}
	}
	if err := mapFunctionTypes(&canonical, func(value hir.Type) (hir.Type, error) {
		if !validStableType(string(value)) {
			return hir.TypeInvalid, fmt.Errorf("invalid stable body type %q", value)
		}
		return value, nil
	}); err != nil {
		return err
	}
	if err := mapFunctionSpans(&canonical, func(span hir.Span) (hir.Span, error) {
		if span.Package != hir.DefaultPackage || span.StartByte < 0 || span.EndByte < span.StartByte {
			return hir.Span{}, fmt.Errorf("stable function body retains an invalid local span %+v", span)
		}
		return span, nil
	}); err != nil {
		return err
	}
	if err := mapFunctionEntities(&canonical, func(value string) (string, error) {
		if strings.Contains(value, "$p") {
			return "", fmt.Errorf("stable function body entity retains a build-local package prefix")
		}
		return value, nil
	}); err != nil {
		return err
	}
	descriptor := FunctionDescriptor{Package: body.Package, Name: body.Function.Name}
	for _, parameterType := range body.Function.ParameterTypes {
		descriptor.ParameterTypes = append(descriptor.ParameterTypes, string(parameterType))
	}
	descriptor.ResultType = string(body.Function.ResultType)
	for _, effect := range body.Function.Effects {
		descriptor.Effects = append(descriptor.Effects, string(effect))
	}
	descriptor.DropGlueFor = string(body.Function.DropGlueFor)
	computed, err := ComputeFunctionID(descriptor)
	if err != nil {
		return fmt.Errorf("verify stable body function descriptor: %w", err)
	}
	if computed != body.ID {
		return fmt.Errorf("stable function-body id does not match its signature")
	}
	return nil
}

func bodyShape(body StableBodyArtifact) (bodyGraphShape, error) {
	maxInt := uint64(^uint(0) >> 1)
	if body.LocalCount > maxInt || body.BlockCount == 0 || body.BlockCount > maxInt {
		return bodyGraphShape{}, fmt.Errorf("stable function-body graph counts exceed host limits")
	}
	return bodyGraphShape{LocalCount: int(body.LocalCount), BlockCount: int(body.BlockCount)}, nil
}

func packageForInstance(instance packagegraph.InstanceID, identities map[hir.PackageID]packagegraph.InstanceID) (hir.PackageID, error) {
	var result hir.PackageID
	found := false
	for packageID, candidate := range identities {
		if candidate != instance {
			continue
		}
		if found && result != packageID {
			return 0, fmt.Errorf("PackageInstanceId %s maps to multiple local packages", instance)
		}
		result = packageID
		found = true
	}
	if !found {
		return 0, fmt.Errorf("PackageInstanceId %s has no local package", instance)
	}
	return result, nil
}

func mapFunctionTypes(function *mir.Function, transform func(hir.Type) (hir.Type, error)) error {
	mapType := func(value *hir.Type) error {
		if *value == "" {
			return nil
		}
		mapped, err := transform(*value)
		if err != nil {
			return err
		}
		*value = mapped
		return nil
	}
	for index := range function.ParameterTypes {
		if err := mapType(&function.ParameterTypes[index]); err != nil {
			return err
		}
	}
	if err := mapType(&function.ResultType); err != nil {
		return err
	}
	if err := mapType(&function.DropGlueFor); err != nil {
		return err
	}
	for index := range function.Resources {
		if err := mapType(&function.Resources[index].Type); err != nil {
			return err
		}
	}
	mapOperand := func(value *mir.Operand) error { return mapType(&value.Type) }
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			for _, value := range []*hir.Type{&operation.Type, &operation.OwnerType, &operation.IndexType} {
				if err := mapType(value); err != nil {
					return err
				}
			}
			for operandIndex := range operation.Operands {
				if err := mapOperand(&operation.Operands[operandIndex]); err != nil {
					return err
				}
			}
		}
		terminator := &function.Blocks[blockIndex].Terminator
		if err := mapType(&terminator.SwitchType); err != nil {
			return err
		}
		if err := mapOperand(&terminator.ReturnValue); err != nil {
			return err
		}
		for operandIndex := range terminator.Arguments {
			if err := mapOperand(&terminator.Arguments[operandIndex]); err != nil {
				return err
			}
		}
		for operandIndex := range terminator.SwitchOperands {
			if err := mapOperand(&terminator.SwitchOperands[operandIndex]); err != nil {
				return err
			}
		}
	}
	return nil
}

func mapFunctionSpans(function *mir.Function, transform func(hir.Span) (hir.Span, error)) error {
	mapSpan := func(value *hir.Span) error {
		mapped, err := transform(*value)
		if err != nil {
			return err
		}
		*value = mapped
		return nil
	}
	mapActions := func(values []mir.CleanupAction) error {
		for index := range values {
			if err := mapSpan(&values[index].Binding); err != nil {
				return err
			}
		}
		return nil
	}
	if err := mapSpan(&function.Span); err != nil {
		return err
	}
	for index := range function.Resources {
		if err := mapSpan(&function.Resources[index].Binding); err != nil {
			return err
		}
	}
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if err := mapSpan(&operation.Span); err != nil {
				return err
			}
			if err := mapSpan(&operation.UnsafeScope); err != nil {
				return err
			}
		}
		terminator := &function.Blocks[blockIndex].Terminator
		if err := mapSpan(&terminator.Span); err != nil {
			return err
		}
		if err := mapSpan(&terminator.UnsafeScope); err != nil {
			return err
		}
	}
	for index := range function.OrderedEvaluations {
		fact := &function.OrderedEvaluations[index]
		if err := mapSpan(&fact.Expression); err != nil {
			return err
		}
		for sequenceIndex := range fact.Sequence {
			if err := mapSpan(&fact.Sequence[sequenceIndex]); err != nil {
				return err
			}
		}
	}
	for index := range function.Calls {
		fact := &function.Calls[index]
		if err := mapSpan(&fact.Call); err != nil {
			return err
		}
		if err := mapSpan(&fact.UnsafeScope); err != nil {
			return err
		}
		for argumentIndex := range fact.ArgumentOrder {
			if err := mapSpan(&fact.ArgumentOrder[argumentIndex]); err != nil {
				return err
			}
		}
	}
	for index := range function.Moves {
		if err := mapSpan(&function.Moves[index].Binding); err != nil {
			return err
		}
		if err := mapSpan(&function.Moves[index].Operand); err != nil {
			return err
		}
	}
	for index := range function.Reassignments {
		fact := &function.Reassignments[index]
		for _, value := range []*hir.Span{&fact.Binding, &fact.Target, &fact.Value} {
			if err := mapSpan(value); err != nil {
				return err
			}
		}
	}
	for index := range function.RegionScopes {
		fact := &function.RegionScopes[index]
		if err := mapSpan(&fact.Region); err != nil {
			return err
		}
		if err := mapSpan(&fact.Reset); err != nil {
			return err
		}
		for placementIndex := range fact.Placements {
			if err := mapSpan(&fact.Placements[placementIndex]); err != nil {
				return err
			}
		}
		if err := mapActions(fact.CleanupBeforeReset); err != nil {
			return err
		}
		for resetIndex := range fact.EarlyResets {
			reset := &fact.EarlyResets[resetIndex]
			if err := mapSpan(&reset.Exit); err != nil {
				return err
			}
			if err := mapSpan(&reset.Reset); err != nil {
				return err
			}
			if err := mapActions(reset.CleanupBeforeReset); err != nil {
				return err
			}
		}
	}
	for index := range function.CleanupPaths {
		if err := mapSpan(&function.CleanupPaths[index].Exit); err != nil {
			return err
		}
		if err := mapActions(function.CleanupPaths[index].Actions); err != nil {
			return err
		}
	}
	for index := range function.ResultBranches {
		fact := &function.ResultBranches[index]
		if err := mapSpan(&fact.Question); err != nil {
			return err
		}
		if err := mapSpan(&fact.ResultExpression); err != nil {
			return err
		}
		if err := mapActions(fact.ErrorCleanupActions); err != nil {
			return err
		}
	}
	for index := range function.TrapPaths {
		if err := mapSpan(&function.TrapPaths[index].Operation); err != nil {
			return err
		}
		if err := mapActions(function.TrapPaths[index].CleanupActions); err != nil {
			return err
		}
	}
	return nil
}

func mapFunctionEntities(function *mir.Function, transform func(string) (string, error)) error {
	mapString := func(value *string) error {
		mapped, err := transform(*value)
		if err != nil {
			return err
		}
		*value = mapped
		return nil
	}
	if err := mapString(&function.ReturnEntity); err != nil {
		return err
	}
	for index := range function.Entities {
		if err := mapString(&function.Entities[index]); err != nil {
			return err
		}
	}
	for index := range function.Resources {
		if err := mapString(&function.Resources[index].Entity); err != nil {
			return err
		}
	}
	for index := range function.DropFlags {
		if err := mapString(&function.DropFlags[index].Entity); err != nil {
			return err
		}
	}
	for index := range function.Loans {
		if err := mapString(&function.Loans[index].Entity); err != nil {
			return err
		}
		if err := mapString(&function.Loans[index].OriginEntity); err != nil {
			return err
		}
	}
	for index := range function.ReturnOrigins {
		if err := mapString(&function.ReturnOrigins[index].Entity); err != nil {
			return err
		}
	}
	for index := range function.RegionValues {
		for _, value := range []*string{&function.RegionValues[index].Entity, &function.RegionValues[index].OperationEntity, &function.RegionValues[index].RegionEntity} {
			if err := mapString(value); err != nil {
				return err
			}
		}
	}
	for index := range function.UnsafeOperations {
		for _, value := range []*string{&function.UnsafeOperations[index].Entity, &function.UnsafeOperations[index].SourceSpanEntity, &function.UnsafeOperations[index].ScopeEntity} {
			if err := mapString(value); err != nil {
				return err
			}
		}
	}
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		if err := mapString(&block.Entity); err != nil {
			return err
		}
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			for _, value := range []*string{&operation.Entity, &operation.Destination, &operation.ProjectionEntity, &operation.DropFlagEntity} {
				if err := mapString(value); err != nil {
					return err
				}
			}
		}
		if err := mapString(&block.Terminator.Entity); err != nil {
			return err
		}
	}
	return nil
}

var sourceEntityKinds = []string{
	"byte-slice-data-address", "call", "cstr-data-address", "cstr-from-raw-parts", "foreign",
	"raw-address", "raw-borrow-pointer", "span", "str-data-address", "str-from-raw-parts", "unsafe-scope",
}

func stabilizeSourceEntity(value string, owner hir.PackageID) (string, error) {
	kind, path, suffix, found := sourceEntityParts(value)
	if !found {
		return value, nil
	}
	packageID, sourcePath := hir.SourcePathParts(path)
	if packageID != owner {
		return "", fmt.Errorf("source entity %q belongs to package %d, want %d", value, packageID, owner)
	}
	return kind + ":@/" + sourcePath + suffix, nil
}

func rebindSourceEntity(value string, owner hir.PackageID) (string, error) {
	kind, path, suffix, found := sourceEntityParts(value)
	if !found {
		return value, nil
	}
	if !strings.HasPrefix(path, "@/") || len(path) == 2 {
		return "", fmt.Errorf("stable source entity %q has no package-neutral path", value)
	}
	return kind + ":" + hir.QualifiedSourcePath(owner, strings.TrimPrefix(path, "@/")) + suffix, nil
}

func sourceEntityParts(value string) (kind, path, suffix string, found bool) {
	first := strings.IndexByte(value, ':')
	last := strings.LastIndexByte(value, ':')
	if first <= 0 || last <= first {
		return "", "", "", false
	}
	previous := strings.LastIndexByte(value[:last], ':')
	if previous <= first {
		return "", "", "", false
	}
	kind = value[:first]
	if !slices.Contains(sourceEntityKinds, kind) {
		return "", "", "", false
	}
	if _, err := strconv.Atoi(value[previous+1 : last]); err != nil {
		return "", "", "", false
	}
	if _, err := strconv.Atoi(value[last+1:]); err != nil {
		return "", "", "", false
	}
	return kind, value[first+1 : previous], value[previous:], true
}

func emptySpan(span hir.Span) bool {
	return span.Package == hir.DefaultPackage && span.Source == "" && span.StartByte == 0 && span.EndByte == 0
}
