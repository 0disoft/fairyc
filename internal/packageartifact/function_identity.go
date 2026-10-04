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
	if function.DropGlueFor != "" {
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
	if builtinType(hir.Type(value)) {
		return true
	}
	for _, prefix := range []string{"ref mut ", "ref ", "ptr mut ", "ptr "} {
		if strings.HasPrefix(value, prefix) {
			return validStableType(strings.TrimPrefix(value, prefix))
		}
	}
	if element, ok := hir.OptionElement(hir.Type(value)); ok {
		return validStableType(string(element))
	}
	if success, failure, ok := hir.ResultElements(hir.Type(value)); ok {
		return validStableType(string(success)) && validStableType(string(failure))
	}
	if strings.HasSuffix(value, "[]") {
		element := strings.TrimSuffix(value, "[]")
		return strings.HasPrefix(element, "pkg[") && !strings.HasSuffix(element, "]") && validStableType(element)
	}
	if strings.HasSuffix(value, "]") {
		if separator := strings.LastIndexByte(value, '['); separator > 0 {
			length, err := strconv.ParseUint(value[separator+1:len(value)-1], 10, 64)
			if err == nil && length > 0 && validStableType(value[:separator]) {
				return true
			}
		}
	}
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
	if element, nominal := hir.StructSliceElement(value); nominal {
		stable, err := stableType(element, coordinates)
		if err != nil {
			return "", err
		}
		return stable + "[]", nil
	}
	if builtinType(value) {
		return string(value), nil
	}
	if element, mutable, ok := hir.BorrowElement(value); ok {
		stable, err := stableType(element, coordinates)
		if err != nil {
			return "", err
		}
		if mutable {
			return "ref mut " + stable, nil
		}
		return "ref " + stable, nil
	}
	if element, mutable, ok := hir.RawPointerElement(value); ok {
		stable, err := stableType(element, coordinates)
		if err != nil {
			return "", err
		}
		if mutable {
			return "ptr mut " + stable, nil
		}
		return "ptr " + stable, nil
	}
	if element, length, ok := hir.FixedArrayParts(value); ok {
		stable, err := stableType(element, coordinates)
		if err != nil {
			return "", err
		}
		return stable + "[" + strconv.FormatUint(length, 10) + "]", nil
	}
	if element, ok := hir.OptionElement(value); ok {
		stable, err := stableType(element, coordinates)
		if err != nil {
			return "", err
		}
		return "Option<" + stable + ">", nil
	}
	if success, failure, ok := hir.ResultElements(value); ok {
		stableSuccess, err := stableType(success, coordinates)
		if err != nil {
			return "", err
		}
		stableFailure, err := stableType(failure, coordinates)
		if err != nil {
			return "", err
		}
		return "Result<" + stableSuccess + ", " + stableFailure + ">", nil
	}
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
	if !validStableType(value) {
		return hir.TypeInvalid, fmt.Errorf("invalid stable type %q", value)
	}
	return rebindValidatedStableType(value, coordinates)
}

func rebindValidatedStableType(value string, coordinates map[hir.PackageID]packagegraph.CoordinateID) (hir.Type, error) {
	if builtinType(hir.Type(value)) {
		return hir.Type(value), nil
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
		element, err := rebindValidatedStableType(strings.TrimPrefix(value, prefix.stable), coordinates)
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
	if element, ok := hir.OptionElement(hir.Type(value)); ok {
		rebound, err := rebindValidatedStableType(string(element), coordinates)
		if err != nil {
			return hir.TypeInvalid, err
		}
		return hir.Option(rebound), nil
	}
	if success, failure, ok := hir.ResultElements(hir.Type(value)); ok {
		reboundSuccess, err := rebindValidatedStableType(string(success), coordinates)
		if err != nil {
			return hir.TypeInvalid, err
		}
		reboundFailure, err := rebindValidatedStableType(string(failure), coordinates)
		if err != nil {
			return hir.TypeInvalid, err
		}
		return hir.Result(reboundSuccess, reboundFailure), nil
	}
	if strings.HasSuffix(value, "[]") {
		element, err := rebindValidatedStableType(strings.TrimSuffix(value, "[]"), coordinates)
		if err != nil {
			return hir.TypeInvalid, err
		}
		return hir.Type("[" + string(element) + "]"), nil
	}
	if strings.HasSuffix(value, "]") {
		separator := strings.LastIndexByte(value, '[')
		if separator > 0 {
			length, err := strconv.ParseUint(value[separator+1:len(value)-1], 10, 64)
			if err != nil || length == 0 {
				return hir.TypeInvalid, fmt.Errorf("invalid stable fixed-array type %q", value)
			}
			element, err := rebindValidatedStableType(value[:separator], coordinates)
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
	var packageID hir.PackageID
	found := false
	for candidate, candidateCoordinate := range coordinates {
		if candidateCoordinate != coordinate {
			continue
		}
		if found && candidate != packageID {
			return hir.TypeInvalid, fmt.Errorf("stable type %q maps PackageCoordinateId %s to multiple local packages", value, coordinate)
		}
		packageID = candidate
		found = true
	}
	if !found {
		return hir.TypeInvalid, fmt.Errorf("stable type %q references unknown PackageCoordinateId %s", value, coordinate)
	}
	logicalName := value[separator+3:]
	return hir.Type(hir.QualifiedModulePath(packageID, logicalName)), nil
}

func nominalTypeParts(value hir.Type) (hir.PackageID, string, bool) {
	text := string(value)
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
	return value == hir.TypeVoid || value == hir.TypeBool || value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeI64 ||
		value == hir.TypeU32 || value == hir.TypeUSize || integerSlice || value == hir.TypeStr ||
		value == hir.TypeCStr || value == hir.TypeRegion || hir.IsCScalar(value)
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
