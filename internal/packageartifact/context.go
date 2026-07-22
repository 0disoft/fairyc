package packageartifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	ContextSchema    = "fairyc.package-type-context/v1"
	ContextVersion   = "fairyc-package-type-context/1"
	ContextIDPrefix  = "fyctx1-sha256-"
	ContextAlgorithm = "sha256"
)

var (
	ErrContextArtifactTooLarge = errors.New("stable type-context artifact exceeds the caller byte limit")
	ErrContextArtifactEncoding = errors.New("stable type-context artifact is not canonically encoded")
)

// ContextID is the full digest of one canonical stable type-context artifact.
type ContextID [sha256.Size]byte

func (id ContextID) String() string {
	return ContextIDPrefix + hex.EncodeToString(id[:])
}

func (id ContextID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *ContextID) UnmarshalText(text []byte) error {
	parsed, err := ParseContextID(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// ParseContextID accepts only the full lowercase canonical representation.
func ParseContextID(value string) (ContextID, error) {
	var result ContextID
	if !strings.HasPrefix(value, ContextIDPrefix) {
		return result, fmt.Errorf("type-context id has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, ContextIDPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("type-context id must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode type-context id: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// ContextDescriptor is the build-number-independent semantic projection of MIR type and foreign
// declaration context. Source spans, entities, PackageID values, and SymbolID values are excluded.
type ContextDescriptor struct {
	Structs          []ContextStruct          `json:"structs"`
	Enums            []ContextEnum            `json:"enums"`
	Resources        []ContextResource        `json:"resources"`
	ForeignFunctions []ContextForeignFunction `json:"foreignFunctions"`
}

type ContextStruct struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	ReprC  bool           `json:"reprC"`
	Fields []ContextField `json:"fields"`
}

type ContextEnum struct {
	Type     string           `json:"type"`
	Name     string           `json:"name"`
	Variants []ContextVariant `json:"variants"`
}

type ContextVariant struct {
	Name   string         `json:"name"`
	Fields []ContextField `json:"fields"`
}

type ContextResource struct {
	Type           string         `json:"type"`
	Fields         []ContextField `json:"fields"`
	DropKind       string         `json:"dropKind"`
	DropFunctionID FunctionID     `json:"dropFunctionId"`
	DropEffects    []string       `json:"dropEffects"`
}

type ContextForeignFunction struct {
	Package        packagegraph.InstanceID `json:"package"`
	Name           string                  `json:"name"`
	LinkName       string                  `json:"linkName"`
	ABI            string                  `json:"abi"`
	ParameterTypes []string                `json:"parameterTypes"`
	ResultType     string                  `json:"resultType"`
}

type ContextField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type contextEnvelope struct {
	Schema  string            `json:"schema"`
	Version string            `json:"version"`
	Context ContextDescriptor `json:"context"`
}

// DescribeContext projects one function-free MIR context into stable package and function
// identities. bindings resolves non-empty resource drop symbols to package-stable FunctionIDs.
func DescribeContext(context mir.Program, identities map[hir.PackageID]packagegraph.InstanceID, bindings map[hir.SymbolID]FunctionID) (ContextDescriptor, error) {
	if len(context.Functions) != 0 {
		return ContextDescriptor{}, fmt.Errorf("stable type-context source contains executable functions")
	}
	result := ContextDescriptor{}
	for _, value := range context.StructTypes {
		typeName, err := stableType(value.Type, identities)
		if err != nil {
			return ContextDescriptor{}, fmt.Errorf("struct %q type: %w", value.Name, err)
		}
		item := ContextStruct{Type: typeName, Name: value.Name, ReprC: value.ReprC}
		for _, field := range value.Fields {
			stable, stableErr := stableType(field.Type, identities)
			if stableErr != nil {
				return ContextDescriptor{}, fmt.Errorf("struct %q field %q: %w", value.Name, field.Name, stableErr)
			}
			item.Fields = append(item.Fields, ContextField{Name: field.Name, Type: stable})
		}
		result.Structs = append(result.Structs, item)
	}
	for _, value := range context.EnumTypes {
		typeName, err := stableType(value.Type, identities)
		if err != nil {
			return ContextDescriptor{}, fmt.Errorf("enum %q type: %w", value.Name, err)
		}
		item := ContextEnum{Type: typeName, Name: value.Name}
		for _, variant := range value.Variants {
			stableVariant := ContextVariant{Name: variant.Name}
			for _, field := range variant.Fields {
				stable, stableErr := stableType(field.Type, identities)
				if stableErr != nil {
					return ContextDescriptor{}, fmt.Errorf("enum %q variant %q field %q: %w", value.Name, variant.Name, field.Name, stableErr)
				}
				stableVariant.Fields = append(stableVariant.Fields, ContextField{Name: field.Name, Type: stable})
			}
			item.Variants = append(item.Variants, stableVariant)
		}
		result.Enums = append(result.Enums, item)
	}
	for _, value := range context.ResourceTypes {
		typeName, err := stableType(value.Type, identities)
		if err != nil {
			return ContextDescriptor{}, fmt.Errorf("resource %q type: %w", value.Type, err)
		}
		item := ContextResource{Type: typeName, DropKind: string(value.DropKind)}
		for _, field := range value.Fields {
			stable, stableErr := stableType(field.Type, identities)
			if stableErr != nil {
				return ContextDescriptor{}, fmt.Errorf("resource %q field %q: %w", value.Type, field.Name, stableErr)
			}
			item.Fields = append(item.Fields, ContextField{Name: field.Name, Type: stable})
		}
		for _, effect := range value.DropEffects {
			item.DropEffects = append(item.DropEffects, string(effect))
		}
		if value.DropSymbol != hir.NoSymbol {
			id, found := bindings[value.DropSymbol]
			if !found {
				return ContextDescriptor{}, fmt.Errorf("resource %q drop symbol %d has no stable function binding", value.Type, value.DropSymbol)
			}
			item.DropFunctionID = id
		}
		result.Resources = append(result.Resources, item)
	}
	for _, value := range context.ForeignFunctions {
		instance, found := identities[value.Span.Package]
		if !found {
			return ContextDescriptor{}, fmt.Errorf("foreign function %q owner package %d has no PackageInstanceId", value.Name, value.Span.Package)
		}
		name, err := logicalFunctionName(value.Name, value.Span.Package)
		if err != nil {
			return ContextDescriptor{}, err
		}
		item := ContextForeignFunction{Package: instance, Name: name, LinkName: value.LinkName, ABI: value.ABI}
		for _, parameterType := range value.ParameterTypes {
			stable, stableErr := stableType(parameterType, identities)
			if stableErr != nil {
				return ContextDescriptor{}, fmt.Errorf("foreign function %q parameter type: %w", value.Name, stableErr)
			}
			item.ParameterTypes = append(item.ParameterTypes, stable)
		}
		item.ResultType, err = stableType(value.ResultType, identities)
		if err != nil {
			return ContextDescriptor{}, fmt.Errorf("foreign function %q result type: %w", value.Name, err)
		}
		result.ForeignFunctions = append(result.ForeignFunctions, item)
	}
	sortContextDescriptor(&result)
	if err := ValidateContextDescriptor(result); err != nil {
		return ContextDescriptor{}, err
	}
	return result, nil
}

// EncodeContextDescriptor returns deterministic, exact-version JSON under a mandatory byte bound.
func EncodeContextDescriptor(value ContextDescriptor, maxBytes uint64) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("stable type-context artifact byte limit must be positive")
	}
	value = canonicalContextDescriptor(value)
	if err := ValidateContextDescriptor(value); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(contextEnvelope{Schema: ContextSchema, Version: ContextVersion, Context: value})
	if err != nil {
		return nil, fmt.Errorf("encode stable type-context artifact: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrContextArtifactTooLarge
	}
	return encoded, nil
}

// DecodeContextDescriptor accepts only the canonical exact-version envelope and validates every
// stable name, type, effect, ordering rule, and drop-function relation before returning it.
func DecodeContextDescriptor(encoded []byte, maxBytes uint64) (ContextDescriptor, error) {
	if maxBytes == 0 {
		return ContextDescriptor{}, fmt.Errorf("stable type-context artifact byte limit must be positive")
	}
	if len(encoded) == 0 {
		return ContextDescriptor{}, fmt.Errorf("stable type-context artifact is empty")
	}
	if uint64(len(encoded)) > maxBytes {
		return ContextDescriptor{}, ErrContextArtifactTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope contextEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return ContextDescriptor{}, fmt.Errorf("decode stable type-context artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return ContextDescriptor{}, fmt.Errorf("decode stable type-context artifact: trailing JSON value")
		}
		return ContextDescriptor{}, fmt.Errorf("decode stable type-context artifact trailing data: %w", err)
	}
	if envelope.Schema != ContextSchema {
		return ContextDescriptor{}, fmt.Errorf("stable type-context schema %q is incompatible with %q", envelope.Schema, ContextSchema)
	}
	if envelope.Version != ContextVersion {
		return ContextDescriptor{}, fmt.Errorf("stable type-context version %q is incompatible with %q", envelope.Version, ContextVersion)
	}
	if err := ValidateContextDescriptor(envelope.Context); err != nil {
		return ContextDescriptor{}, fmt.Errorf("verify decoded stable type-context artifact: %w", err)
	}
	canonical, err := EncodeContextDescriptor(envelope.Context, maxBytes)
	if err != nil {
		return ContextDescriptor{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return ContextDescriptor{}, ErrContextArtifactEncoding
	}
	return envelope.Context, nil
}

// ComputeContextID hashes the canonical exact-version context artifact.
func ComputeContextID(value ContextDescriptor, maxBytes uint64) (ContextID, error) {
	encoded, err := EncodeContextDescriptor(value, maxBytes)
	if err != nil {
		return ContextID{}, err
	}
	return sha256.Sum256(encoded), nil
}

// ValidateContextDescriptor checks the stable semantic projection without consulting process-local
// MIR state.
func ValidateContextDescriptor(value ContextDescriptor) error {
	seenTypes := make(map[string]string, len(value.Structs)+len(value.Enums)+len(value.Resources))
	for index, item := range value.Structs {
		if index > 0 && value.Structs[index-1].Type >= item.Type {
			return fmt.Errorf("stable type-context structs are not sorted and unique")
		}
		if !validNamedType(item.Type, item.Name) || item.ReprC && len(item.Fields) == 0 {
			return fmt.Errorf("stable type-context has an invalid struct %q", item.Type)
		}
		if err := recordContextType(seenTypes, item.Type, "struct"); err != nil {
			return err
		}
		if err := validateContextFields(item.Fields); err != nil {
			return fmt.Errorf("stable struct %q: %w", item.Type, err)
		}
	}
	for index, item := range value.Enums {
		if index > 0 && value.Enums[index-1].Type >= item.Type {
			return fmt.Errorf("stable type-context enums are not sorted and unique")
		}
		if !validNamedType(item.Type, item.Name) {
			return fmt.Errorf("stable type-context has an invalid enum %q", item.Type)
		}
		if err := recordContextType(seenTypes, item.Type, "enum"); err != nil {
			return err
		}
		seenVariants := make(map[string]struct{}, len(item.Variants))
		for _, variant := range item.Variants {
			if !source.ValidIdentifier(variant.Name) {
				return fmt.Errorf("stable enum %q has an invalid variant", item.Type)
			}
			if _, duplicate := seenVariants[variant.Name]; duplicate {
				return fmt.Errorf("stable enum %q has a duplicate variant %q", item.Type, variant.Name)
			}
			seenVariants[variant.Name] = struct{}{}
			if err := validateContextFields(variant.Fields); err != nil {
				return fmt.Errorf("stable enum %q variant %q: %w", item.Type, variant.Name, err)
			}
		}
	}
	for index, item := range value.Resources {
		if index > 0 && value.Resources[index-1].Type >= item.Type {
			return fmt.Errorf("stable type-context resources are not sorted and unique")
		}
		if !validStableNominalType(item.Type) {
			return fmt.Errorf("stable type-context has an invalid resource %q", item.Type)
		}
		if err := recordContextType(seenTypes, item.Type, "resource"); err != nil {
			return err
		}
		if err := validateContextFields(item.Fields); err != nil {
			return fmt.Errorf("stable resource %q: %w", item.Type, err)
		}
		if item.DropKind != string(hir.ResourceDropEmpty) && item.DropKind != string(hir.ResourceDropNonEmpty) {
			return fmt.Errorf("stable resource %q has unsupported drop kind %q", item.Type, item.DropKind)
		}
		if item.DropKind == string(hir.ResourceDropEmpty) && item.DropFunctionID != (FunctionID{}) ||
			item.DropKind == string(hir.ResourceDropNonEmpty) && item.DropFunctionID == (FunctionID{}) {
			return fmt.Errorf("stable resource %q has an inconsistent drop function", item.Type)
		}
		if item.DropKind == string(hir.ResourceDropEmpty) && len(item.DropEffects) != 0 {
			return fmt.Errorf("stable resource %q has effects on an empty drop", item.Type)
		}
		if !slices.IsSorted(item.DropEffects) {
			return fmt.Errorf("stable resource %q effects are not sorted", item.Type)
		}
		for effectIndex, effect := range item.DropEffects {
			if _, ok := hir.ParseEffect(effect); !ok || effectIndex > 0 && item.DropEffects[effectIndex-1] == effect {
				return fmt.Errorf("stable resource %q effects are unknown or duplicated", item.Type)
			}
		}
	}
	foreignByLinkName := make(map[string]ContextForeignFunction, len(value.ForeignFunctions))
	for index, item := range value.ForeignFunctions {
		if index > 0 && compareContextForeign(value.ForeignFunctions[index-1], item) >= 0 {
			return fmt.Errorf("stable type-context foreign functions are not sorted and unique")
		}
		if item.Package == (packagegraph.InstanceID{}) || !validLogicalFunctionName(item.Name) ||
			item.LinkName == "" || !utf8.ValidString(item.LinkName) || item.ABI != "C" || !validStableType(item.ResultType) {
			return fmt.Errorf("stable type-context has an invalid foreign function %q", item.Name)
		}
		if stableBorrowType(item.ResultType) {
			return fmt.Errorf("stable foreign function %q result exposes a safe borrow", item.Name)
		}
		for _, parameterType := range item.ParameterTypes {
			if !validStableType(parameterType) {
				return fmt.Errorf("stable foreign function %q has an invalid parameter type", item.Name)
			}
			if stableBorrowType(parameterType) {
				return fmt.Errorf("stable foreign function %q parameter exposes a safe borrow", item.Name)
			}
		}
		if previous, duplicate := foreignByLinkName[item.LinkName]; duplicate &&
			(previous.ABI != item.ABI || previous.ResultType != item.ResultType || !slices.Equal(previous.ParameterTypes, item.ParameterTypes)) {
			return fmt.Errorf("stable foreign declarations for link name %q have conflicting signatures", item.LinkName)
		}
		foreignByLinkName[item.LinkName] = item
	}
	return nil
}

func recordContextType(seen map[string]string, value, kind string) error {
	if previous, duplicate := seen[value]; duplicate {
		return fmt.Errorf("stable type %q is both %s and %s", value, previous, kind)
	}
	seen[value] = kind
	return nil
}

func validateContextFields(fields []ContextField) error {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if !source.ValidIdentifier(field.Name) || !validStableType(field.Type) {
			return fmt.Errorf("field %q is invalid", field.Name)
		}
		if _, duplicate := seen[field.Name]; duplicate {
			return fmt.Errorf("field %q is duplicated", field.Name)
		}
		seen[field.Name] = struct{}{}
	}
	return nil
}

func validNamedType(value, name string) bool {
	return validStableNominalType(value) && source.ValidIdentifier(name) && strings.HasSuffix(value, "::"+name)
}

func validStableNominalType(value string) bool {
	return strings.HasPrefix(value, "pkg[") && validStableType(value)
}

func stableBorrowType(value string) bool {
	return strings.HasPrefix(value, "ref ") || strings.HasPrefix(value, "ref mut ")
}

func sortContextDescriptor(value *ContextDescriptor) {
	slices.SortFunc(value.Structs, func(left, right ContextStruct) int { return strings.Compare(left.Type, right.Type) })
	slices.SortFunc(value.Enums, func(left, right ContextEnum) int { return strings.Compare(left.Type, right.Type) })
	slices.SortFunc(value.Resources, func(left, right ContextResource) int { return strings.Compare(left.Type, right.Type) })
	slices.SortFunc(value.ForeignFunctions, compareContextForeign)
}

func compareContextForeign(left, right ContextForeignFunction) int {
	if byPackage := bytes.Compare(left.Package[:], right.Package[:]); byPackage != 0 {
		return byPackage
	}
	return strings.Compare(left.Name, right.Name)
}

func canonicalContextDescriptor(value ContextDescriptor) ContextDescriptor {
	value = cloneContextDescriptor(value)
	if len(value.Structs) == 0 {
		value.Structs = nil
	}
	if len(value.Enums) == 0 {
		value.Enums = nil
	}
	if len(value.Resources) == 0 {
		value.Resources = nil
	}
	if len(value.ForeignFunctions) == 0 {
		value.ForeignFunctions = nil
	}
	for index := range value.Structs {
		if len(value.Structs[index].Fields) == 0 {
			value.Structs[index].Fields = nil
		}
	}
	for index := range value.Enums {
		if len(value.Enums[index].Variants) == 0 {
			value.Enums[index].Variants = nil
		}
		for variantIndex := range value.Enums[index].Variants {
			if len(value.Enums[index].Variants[variantIndex].Fields) == 0 {
				value.Enums[index].Variants[variantIndex].Fields = nil
			}
		}
	}
	for index := range value.Resources {
		if len(value.Resources[index].Fields) == 0 {
			value.Resources[index].Fields = nil
		}
		if len(value.Resources[index].DropEffects) == 0 {
			value.Resources[index].DropEffects = nil
		}
	}
	for index := range value.ForeignFunctions {
		if len(value.ForeignFunctions[index].ParameterTypes) == 0 {
			value.ForeignFunctions[index].ParameterTypes = nil
		}
	}
	return value
}

func cloneContextDescriptor(value ContextDescriptor) ContextDescriptor {
	result := value
	result.Structs = slices.Clone(value.Structs)
	for index, item := range value.Structs {
		result.Structs[index].Fields = slices.Clone(item.Fields)
	}
	result.Enums = slices.Clone(value.Enums)
	for index, item := range value.Enums {
		result.Enums[index].Variants = slices.Clone(item.Variants)
		for variantIndex, variant := range item.Variants {
			result.Enums[index].Variants[variantIndex].Fields = slices.Clone(variant.Fields)
		}
	}
	result.Resources = slices.Clone(value.Resources)
	for index, item := range value.Resources {
		result.Resources[index].Fields = slices.Clone(item.Fields)
		result.Resources[index].DropEffects = slices.Clone(item.DropEffects)
	}
	result.ForeignFunctions = slices.Clone(value.ForeignFunctions)
	for index, item := range value.ForeignFunctions {
		result.ForeignFunctions[index].ParameterTypes = slices.Clone(item.ParameterTypes)
	}
	return result
}
