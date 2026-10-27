// Package packageartifact defines compiler-private identities and boundaries for reusable package
// artifacts. Its encodings are versioned independently from source syntax, HIR symbols, and native
// ABI names.
package packageartifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	FunctionIDAlgorithm = "sha256"
	FunctionIDEncoding  = "fairyc.package-function/v2"
	FunctionIDPrefix    = "fyfn2-sha256-"
)

const functionIDDomain = "fairyc.package-function"

// FunctionID is the full digest of one package-stable function descriptor. It is not a HIR
// SymbolID and is not, by itself, a native linker symbol.
type FunctionID [sha256.Size]byte

// String returns the only serialized FunctionID spelling.
func (id FunctionID) String() string {
	return FunctionIDPrefix + hex.EncodeToString(id[:])
}

// MarshalText implements encoding.TextMarshaler for compiler-private artifact metadata.
func (id FunctionID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText accepts only the full lowercase canonical representation.
func (id *FunctionID) UnmarshalText(text []byte) error {
	parsed, err := ParseFunctionID(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// ParseFunctionID parses one canonical full-digest FunctionID.
func ParseFunctionID(value string) (FunctionID, error) {
	var result FunctionID
	if !strings.HasPrefix(value, FunctionIDPrefix) {
		return result, fmt.Errorf("package function id has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, FunctionIDPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("package function id must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode package function id: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// FunctionDescriptor contains the build-number-independent signature facts required to identify
// one package-owned MIR function. Types use an artifact-private stable spelling.
type FunctionDescriptor struct {
	Package        packagegraph.CoordinateID
	Name           string
	ParameterTypes []string
	ResultType     string
	Effects        []string
	DropGlueFor    string
}

// DescribeFunction removes disposable HIR package numbers from one MIR function and every type in
// its signature. coordinates must contain the owner and every package referenced by a nominal type.
func DescribeFunction(function mir.Function, coordinates map[hir.PackageID]packagegraph.CoordinateID) (FunctionDescriptor, error) {
	owner := function.Span.Package
	coordinate, found := coordinates[owner]
	if !found {
		return FunctionDescriptor{}, fmt.Errorf("function %q owner package %d has no PackageCoordinateId", function.Name, owner)
	}
	name, err := logicalFunctionName(function.Name, owner)
	if err != nil {
		return FunctionDescriptor{}, err
	}
	descriptor := FunctionDescriptor{Package: coordinate, Name: name}
	for _, parameterType := range function.ParameterTypes {
		stable, stableErr := stableType(parameterType, coordinates)
		if stableErr != nil {
			return FunctionDescriptor{}, fmt.Errorf("function %q parameter type: %w", function.Name, stableErr)
		}
		descriptor.ParameterTypes = append(descriptor.ParameterTypes, stable)
	}
	descriptor.ResultType, err = stableType(function.ResultType, coordinates)
	if err != nil {
		return FunctionDescriptor{}, fmt.Errorf("function %q result type: %w", function.Name, err)
	}
	if !slices.Equal(function.Effects, hir.CanonicalEffects(function.Effects)) {
		return FunctionDescriptor{}, fmt.Errorf("function %q effects are not canonical", function.Name)
	}
	for _, effect := range function.Effects {
		if _, ok := hir.ParseEffect(string(effect)); !ok {
			return FunctionDescriptor{}, fmt.Errorf("function %q has unknown effect %q", function.Name, effect)
		}
		descriptor.Effects = append(descriptor.Effects, string(effect))
	}
	if function.DropGlueFor != hir.ParseType("") {
		descriptor.DropGlueFor, err = stableType(function.DropGlueFor, coordinates)
		if err != nil {
			return FunctionDescriptor{}, fmt.Errorf("function %q drop-glue type: %w", function.Name, err)
		}
	}
	return descriptor, nil
}

// ComputeFunctionID validates and hashes one canonical descriptor.
func ComputeFunctionID(descriptor FunctionDescriptor) (FunctionID, error) {
	encoded, err := EncodeFunctionDescriptor(descriptor)
	if err != nil {
		return FunctionID{}, err
	}
	return sha256.Sum256(encoded), nil
}

// EncodeFunctionDescriptor returns the exact v2 byte stream hashed by ComputeFunctionID.
func EncodeFunctionDescriptor(descriptor FunctionDescriptor) ([]byte, error) {
	if err := validateFunctionDescriptor(descriptor); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	encoded.WriteString(functionIDDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(2)
	encoded.Write(descriptor.Package[:])
	writeString(&encoded, descriptor.Name)
	writeStrings(&encoded, descriptor.ParameterTypes)
	writeString(&encoded, descriptor.ResultType)
	writeStrings(&encoded, descriptor.Effects)
	writeString(&encoded, descriptor.DropGlueFor)
	return encoded.Bytes(), nil
}

func validateFunctionDescriptor(descriptor FunctionDescriptor) error {
	if descriptor.Package == (packagegraph.CoordinateID{}) {
		return fmt.Errorf("package function descriptor has an empty PackageCoordinateId")
	}
	if !validLogicalFunctionName(descriptor.Name) {
		return fmt.Errorf("package function descriptor has an invalid logical name")
	}
	for _, value := range descriptor.ParameterTypes {
		if !validStableType(value) {
			return fmt.Errorf("package function descriptor has an invalid parameter type")
		}
	}
	if !validStableType(descriptor.ResultType) {
		return fmt.Errorf("package function descriptor has an invalid result type")
	}
	if !slices.IsSorted(descriptor.Effects) {
		return fmt.Errorf("package function descriptor effects are not sorted")
	}
	for index, effect := range descriptor.Effects {
		if _, ok := hir.ParseEffect(effect); !ok || index > 0 && descriptor.Effects[index-1] == effect {
			return fmt.Errorf("package function descriptor effects are unknown or duplicated")
		}
	}
	if descriptor.DropGlueFor != "" && !validStableType(descriptor.DropGlueFor) {
		return fmt.Errorf("package function descriptor has an invalid drop-glue type")
	}
	return nil
}

func validLogicalFunctionName(name string) bool {
	if name == "" || !utf8.ValidString(name) || strings.HasPrefix(name, "$p") {
		return false
	}
	segments := strings.Split(name, "::")
	if len(segments) < 2 {
		return false
	}
	for index, segment := range segments {
		if source.ValidIdentifier(segment) {
			continue
		}
		if index == len(segments)-1 && strings.HasPrefix(segment, "$drop$") && source.ValidIdentifier(strings.TrimPrefix(segment, "$drop$")) {
			continue
		}
		return false
	}
	return true
}

func validStableType(value string) bool {
	if value == "" || !utf8.ValidString(value) || strings.Contains(value, "$p") {
		return false
	}
	// Substrings and HIR child spellings preserve the checked UTF-8 and cannot
	// introduce a build-local prefix. Visit each child without recursive calls.
	pending := []string{value}
	for len(pending) != 0 {
		value = stripStableIndirection(pending[len(pending)-1])
		pending = pending[:len(pending)-1]
		if value == "" {
			return false
		}
		parsed := hir.ParseType(value)
		if builtinType(parsed) {
			continue
		}
		if element, ok := hir.OptionElement(parsed); ok {
			pending = append(pending, element.String())
			continue
		}
		if success, failure, ok := hir.ResultElements(parsed); ok {
			pending = append(pending, failure.String(), success.String())
			continue
		}
		if strings.HasSuffix(value, "[]") {
			element := strings.TrimSuffix(value, "[]")
			if !strings.HasPrefix(element, "pkg[") || strings.HasSuffix(element, "]") {
				return false
			}
			pending = append(pending, element)
			continue
		}
		if strings.HasSuffix(value, "]") {
			if separator := strings.LastIndexByte(value, '['); separator > 0 {
				length, err := strconv.ParseUint(value[separator+1:len(value)-1], 10, 64)
				if err == nil && length > 0 {
					pending = append(pending, value[:separator])
					continue
				}
			}
		}
		if !validStableNominalSpelling(value) {
			return false
		}
	}
	return true
}

func stripStableIndirection(value string) string {
	for {
		switch {
		case strings.HasPrefix(value, "ref mut "), strings.HasPrefix(value, "ptr mut "):
			value = value[8:]
		case strings.HasPrefix(value, "ref "), strings.HasPrefix(value, "ptr "):
			value = value[4:]
		default:
			return value
		}
	}
}

func validStableNominalSpelling(value string) bool {
	const nominalPrefix = "pkg["
	if !strings.HasPrefix(value, nominalPrefix) {
		return false
	}
	separator := strings.Index(value, "]::")
	if separator <= len(nominalPrefix) || separator+3 >= len(value) {
		return false
	}
	if _, err := packagegraph.ParseCoordinateID(value[len(nominalPrefix):separator]); err != nil {
		return false
	}
	for _, segment := range strings.Split(value[separator+3:], "::") {
		if !source.ValidIdentifier(segment) {
			return false
		}
	}
	return true
}

func logicalFunctionName(name string, owner hir.PackageID) (string, error) {
	if owner == hir.DefaultPackage {
		if name == "" || strings.HasPrefix(name, "$p") {
			return "", fmt.Errorf("function %q does not match default package identity", name)
		}
		return name, nil
	}
	prefix := "$p" + strconv.FormatUint(uint64(owner), 10) + "::"
	if !strings.HasPrefix(name, prefix) || len(name) == len(prefix) {
		return "", fmt.Errorf("function %q does not match owner package %d", name, owner)
	}
	return strings.TrimPrefix(name, prefix), nil
}

func stableType(value hir.Type, coordinates map[hir.PackageID]packagegraph.CoordinateID) (string, error) {
	// Preserve the allocation-free path for the common scalar and builtin types.
	if builtinType(value) {
		return value.String(), nil
	}
	if value.Descriptor().Kind == hir.TypeKindNominal {
		return stableNominalType(value, coordinates)
	}
	var output strings.Builder
	if err := writeStableType(&output, value, coordinates); err != nil {
		return "", err
	}
	return output.String(), nil
}

// Write children directly to one buffer. Returning a fresh child string at each
// nesting level copies the same suffix repeatedly and makes deep types quadratic.
func writeStableType(output *strings.Builder, value hir.Type, coordinates map[hir.PackageID]packagegraph.CoordinateID) error {
	if element, nominal := hir.StructSliceElement(value); nominal {
		if err := writeStableType(output, element, coordinates); err != nil {
			return err
		}
		output.WriteString("[]")
		return nil
	}
	if builtinType(value) {
		output.WriteString(value.String())
		return nil
	}
	if element, mutable, ok := hir.BorrowElement(value); ok {
		if mutable {
			output.WriteString("ref mut ")
		} else {
			output.WriteString("ref ")
		}
		return writeStableType(output, element, coordinates)
	}
	if element, mutable, ok := hir.RawPointerElement(value); ok {
		if mutable {
			output.WriteString("ptr mut ")
		} else {
			output.WriteString("ptr ")
		}
		return writeStableType(output, element, coordinates)
	}
	if element, length, ok := hir.FixedArrayParts(value); ok {
		if err := writeStableType(output, element, coordinates); err != nil {
			return err
		}
		output.WriteByte('[')
		output.WriteString(strconv.FormatUint(length, 10))
		output.WriteByte(']')
		return nil
	}
	if element, ok := hir.OptionElement(value); ok {
		output.WriteString("Option<")
		if err := writeStableType(output, element, coordinates); err != nil {
			return err
		}
		output.WriteByte('>')
		return nil
	}
	if success, failure, ok := hir.ResultElements(value); ok {
		output.WriteString("Result<")
		if err := writeStableType(output, success, coordinates); err != nil {
			return err
		}
		output.WriteString(", ")
		if err := writeStableType(output, failure, coordinates); err != nil {
			return err
		}
		output.WriteByte('>')
		return nil
	}
	stable, err := stableNominalType(value, coordinates)
	if err != nil {
		return err
	}
	output.WriteString(stable)
	return nil
}

// Leaf nominal types keep the original direct projection without a builder.
func stableNominalType(value hir.Type, coordinates map[hir.PackageID]packagegraph.CoordinateID) (string, error) {
	packageID, logicalName, ok := nominalTypeParts(value)
	if !ok {
		return "", fmt.Errorf("unsupported canonical type %q", value)
	}
	coordinate, found := coordinates[packageID]
	if !found {
		return "", fmt.Errorf("type %q references package %d without PackageCoordinateId", value, packageID)
	}
	return "pkg[" + coordinate.String() + "]::" + logicalName, nil
}

// rebindStableType resolves one validated artifact-private type spelling into the canonical HIR
// spelling for a fresh build. PackageCoordinateId values must map one-to-one to build-local package
// numbers; an absent or ambiguous identity fails closed instead of guessing a package.
func rebindStableType(value string, coordinates map[hir.PackageID]packagegraph.CoordinateID) (hir.Type, error) {
	return rebindStableTypeWithResolver(value, func(coordinate packagegraph.CoordinateID) (packageCoordinateBinding, bool) {
		return scanPackageCoordinate(coordinate, coordinates)
	})
}

func rebindStableTypeWithResolver(value string, resolve packageResolver) (hir.Type, error) {
	if !validStableType(value) {
		return hir.TypeInvalid, fmt.Errorf("invalid stable type %q", value)
	}
	return rebindValidatedStableType(value, resolve)
}

func rebindValidatedStableType(value string, resolve packageResolver) (hir.Type, error) {
	if builtinType(hir.ParseType(value)) {
		return hir.ParseType(value), nil
	}
	for _, prefix := range []struct {
		stable  string
		mutable bool
		raw     bool
	}{
		{stable: "ref mut ", mutable: true},
		{stable: "ref "},
		{stable: "ptr mut ", mutable: true, raw: true},
		{stable: "ptr ", raw: true},
	} {
		if !strings.HasPrefix(value, prefix.stable) {
			continue
		}
		element, err := rebindValidatedStableType(strings.TrimPrefix(value, prefix.stable), resolve)
		if err != nil {
			return hir.TypeInvalid, err
		}
		if prefix.raw {
			return hir.RawPointer(element, prefix.mutable), nil
		}
		if prefix.mutable {
			return hir.MutableBorrow(element), nil
		}
		return hir.SharedBorrow(element), nil
	}
	if element, ok := hir.OptionElement(hir.ParseType(value)); ok {
		rebound, err := rebindValidatedStableType(element.String(), resolve)
		if err != nil {
			return hir.TypeInvalid, err
		}
		return hir.Option(rebound), nil
	}
	if success, failure, ok := hir.ResultElements(hir.ParseType(value)); ok {
		reboundSuccess, err := rebindValidatedStableType(success.String(), resolve)
		if err != nil {
			return hir.TypeInvalid, err
		}
		reboundFailure, err := rebindValidatedStableType(failure.String(), resolve)
		if err != nil {
			return hir.TypeInvalid, err
		}
		return hir.Result(reboundSuccess, reboundFailure), nil
	}
	if strings.HasSuffix(value, "[]") {
		element, err := rebindValidatedStableType(strings.TrimSuffix(value, "[]"), resolve)
		if err != nil {
			return hir.TypeInvalid, err
		}
		return hir.Slice(element), nil
	}
	if strings.HasSuffix(value, "]") {
		separator := strings.LastIndexByte(value, '[')
		if separator > 0 {
			length, err := strconv.ParseUint(value[separator+1:len(value)-1], 10, 64)
			if err != nil || length == 0 {
				return hir.TypeInvalid, fmt.Errorf("invalid stable fixed-array type %q", value)
			}
			element, err := rebindValidatedStableType(value[:separator], resolve)
			if err != nil {
				return hir.TypeInvalid, err
			}
			return hir.FixedArray(element, length), nil
		}
	}

	const nominalPrefix = "pkg["
	separator := strings.Index(value, "]::")
	coordinate, err := packagegraph.ParseCoordinateID(value[len(nominalPrefix):separator])
	if err != nil {
		return hir.TypeInvalid, fmt.Errorf("parse stable nominal type %q: %w", value, err)
	}
	binding, found := resolve(coordinate)
	if binding.ambiguous {
		return hir.TypeInvalid, fmt.Errorf("stable type %q maps PackageCoordinateId %s to multiple local packages", value, coordinate)
	}
	if !found {
		return hir.TypeInvalid, fmt.Errorf("stable type %q references unknown PackageCoordinateId %s", value, coordinate)
	}
	logicalName := value[separator+3:]
	return hir.ParseType(hir.QualifiedModulePath(binding.id, logicalName)), nil
}

func nominalTypeParts(value hir.Type) (hir.PackageID, string, bool) {
	text := value.String()
	if strings.HasPrefix(text, "$p") {
		separator := strings.Index(text, "::")
		if separator <= 2 || separator+2 >= len(text) {
			return 0, "", false
		}
		parsed, err := strconv.ParseUint(text[2:separator], 10, 32)
		if err != nil || parsed == 0 {
			return 0, "", false
		}
		return hir.PackageID(parsed), text[separator+2:], true
	}
	if !strings.Contains(text, "::") {
		return 0, "", false
	}
	return hir.DefaultPackage, text, true
}

func builtinType(value hir.Type) bool {
	_, integerSlice := hir.ScalarViewElement(value)
	return value == hir.TypeCCallbackI32UserData || value == hir.TypeVoid || value == hir.TypeBool || value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeI64 ||
		value == hir.TypeU32 || value == hir.TypeUSize || integerSlice || value == hir.TypeStr ||
		value == hir.TypeCStr || value == hir.TypeRegion || hir.IsCScalar(value) || hir.IsFloatScalar(value) || hir.IsIntegerTransportScalar(value)
}

func writeStrings(buffer *bytes.Buffer, values []string) {
	writeUint64(buffer, uint64(len(values)))
	for _, value := range values {
		writeString(buffer, value)
	}
}

func writeString(buffer *bytes.Buffer, value string) {
	writeUint64(buffer, uint64(len(value)))
	buffer.WriteString(value)
}

func writeUint64(buffer *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buffer.Write(raw[:])
}
