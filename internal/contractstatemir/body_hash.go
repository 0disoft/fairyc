package contractstatemir

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	StateBodyHashAlgorithm = "sha256"
	StateBodyHashEncoding  = "fairyc.contract-state-package-body/v1"
	StateBodyHashPrefix    = "fycsbody1-sha256-"
)

const stateBodyHashDomain = "fairyc.contract-state-package-body"

// StateBodyHash identifies one package's exact verified transaction bodies
// without retaining build-local package, symbol, or capability identities.
type StateBodyHash [sha256.Size]byte

func (value StateBodyHash) String() string {
	return StateBodyHashPrefix + hex.EncodeToString(value[:])
}

func (value StateBodyHash) MarshalText() ([]byte, error) {
	return []byte(value.String()), nil
}

func (value *StateBodyHash) UnmarshalText(text []byte) error {
	parsed, err := ParseStateBodyHash(string(text))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

// ParseStateBodyHash accepts only the complete lowercase canonical spelling.
func ParseStateBodyHash(value string) (StateBodyHash, error) {
	var result StateBodyHash
	if !strings.HasPrefix(value, StateBodyHashPrefix) {
		return result, fmt.Errorf("contract-state body hash has invalid prefix")
	}
	digest := strings.TrimPrefix(value, StateBodyHashPrefix)
	if len(digest) != sha256.Size*2 || digest != strings.ToLower(digest) {
		return result, fmt.Errorf("contract-state body hash must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil {
		return result, fmt.Errorf("decode contract-state body hash: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

type stableStateTransaction struct {
	function []byte
	program  Program
}

// ComputeHIRStateBodyHashes verifies every transaction once, removes all
// build-local identities, and hashes each requested package independently.
// packageIDs may be in any order but must be unique.
func ComputeHIRStateBodyHashes(input hir.Package, packageIDs []hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.PackageID]StateBodyHash, error) {
	requested := slices.Clone(packageIDs)
	slices.Sort(requested)
	for index := 1; index < len(requested); index++ {
		if requested[index-1] == requested[index] {
			return nil, fmt.Errorf("contract-state body hash package %d is duplicated", requested[index])
		}
	}
	layouts, err := LowerHIRLayouts(input)
	if err != nil {
		return nil, err
	}
	layoutByPackage := make(map[hir.PackageID]Layout, len(layouts))
	for _, item := range layouts {
		layoutByPackage[item.Package] = item.Layout
	}
	lowered, err := LowerHIRTransactions(input)
	if err != nil {
		return nil, err
	}
	transactions := make(map[hir.PackageID][]stableStateTransaction)
	for _, item := range lowered {
		module, function, err := findHIRFunctionOwner(input, item.Package, item.Function)
		if err != nil {
			return nil, err
		}
		descriptor, err := describeStateBodyFunction(module, function, coordinates)
		if err != nil {
			return nil, fmt.Errorf("describe package %d state function %d: %w", item.Package, item.Function, err)
		}
		transactions[item.Package] = append(transactions[item.Package], stableStateTransaction{function: descriptor, program: item.Program})
	}

	result := make(map[hir.PackageID]StateBodyHash, len(requested))
	for _, packageID := range requested {
		coordinate, found := coordinates[packageID]
		if !found || coordinate == (packagegraph.CoordinateID{}) {
			return nil, fmt.Errorf("contract-state body package %d has no PackageCoordinateId", packageID)
		}
		items := transactions[packageID]
		slices.SortFunc(items, func(left, right stableStateTransaction) int {
			return bytes.Compare(left.function, right.function)
		})
		for index := 1; index < len(items); index++ {
			if bytes.Equal(items[index-1].function, items[index].function) {
				return nil, fmt.Errorf("package %d state transactions share one stable function identity", packageID)
			}
		}
		result[packageID] = hashStableStateBody(coordinate, layoutByPackage[packageID], items)
	}
	return result, nil
}

func hashStableStateBody(coordinate packagegraph.CoordinateID, layout Layout, transactions []stableStateTransaction) StateBodyHash {
	hasher := sha256.New()
	hasher.Write([]byte(stateBodyHashDomain))
	hasher.Write([]byte{0, 1})
	hasher.Write(coordinate[:])
	if layout.Hash == ([sha256.Size]byte{}) {
		hasher.Write([]byte{0})
	} else {
		hasher.Write([]byte{1})
		hasher.Write(layout.Hash[:])
	}
	writeStateBodyUint64(hasher, uint64(len(transactions)))
	for _, transaction := range transactions {
		writeStateBodyUint64(hasher, uint64(len(transaction.function)))
		hasher.Write(transaction.function)
		writeStateBodyStrings(hasher, transaction.program.Effects)
		writeStateBodyUint64(hasher, uint64(len(transaction.program.Operations)))
		for _, operation := range transaction.program.Operations {
			writeStateBodyString(hasher, string(operation.Kind))
			writeStateBodyUint64(hasher, uint64(operation.FieldIndex))
			writeStateBodyValue(hasher, operation.Key)
			writeStateBodyValue(hasher, operation.Value)
		}
		writeStateBodyString(hasher, string(transaction.program.Exit))
		writeStateBodyString(hasher, transaction.program.TrapCategory)
	}
	var result StateBodyHash
	copy(result[:], hasher.Sum(nil))
	return result
}

func describeStateBodyFunction(module hir.Module, function hir.Function, coordinates map[hir.PackageID]packagegraph.CoordinateID) ([]byte, error) {
	if module.Path == "" || function.Name == "" {
		return nil, fmt.Errorf("state function has an empty logical name")
	}
	logicalName := module.Path + "::" + function.Name
	if !validStateBodyLogicalName(logicalName) {
		return nil, fmt.Errorf("state function has invalid logical name %q", logicalName)
	}
	if !slices.Equal(function.Effects, hir.CanonicalEffects(function.Effects)) {
		return nil, fmt.Errorf("state function %q effects are not canonical", logicalName)
	}
	var encoded bytes.Buffer
	writeStateBodyString(&encoded, logicalName)
	writeStateBodyUint64(&encoded, uint64(len(function.Parameters)))
	for _, parameter := range function.Parameters {
		stable, err := stableStateBodyType(parameter.Type, coordinates)
		if err != nil {
			return nil, fmt.Errorf("state function %q parameter type: %w", logicalName, err)
		}
		writeStateBodyString(&encoded, stable)
	}
	result, err := stableStateBodyType(function.ResultType, coordinates)
	if err != nil {
		return nil, fmt.Errorf("state function %q result type: %w", logicalName, err)
	}
	writeStateBodyString(&encoded, result)
	writeStateBodyStrings(&encoded, function.Effects)
	if function.DropGlueFor == "" {
		writeStateBodyString(&encoded, "")
	} else {
		stable, err := stableStateBodyType(function.DropGlueFor, coordinates)
		if err != nil {
			return nil, fmt.Errorf("state function %q drop-glue type: %w", logicalName, err)
		}
		writeStateBodyString(&encoded, stable)
	}
	return encoded.Bytes(), nil
}

func stableStateBodyType(value hir.Type, coordinates map[hir.PackageID]packagegraph.CoordinateID) (string, error) {
	if stateBodyBuiltinType(value) {
		return string(value), nil
	}
	if element, mutable, ok := hir.BorrowElement(value); ok {
		stable, err := stableStateBodyType(element, coordinates)
		if err != nil {
			return "", err
		}
		if mutable {
			return "ref mut " + stable, nil
		}
		return "ref " + stable, nil
	}
	if element, mutable, ok := hir.RawPointerElement(value); ok {
		stable, err := stableStateBodyType(element, coordinates)
		if err != nil {
			return "", err
		}
		if mutable {
			return "ptr mut " + stable, nil
		}
		return "ptr " + stable, nil
	}
	if element, length, ok := hir.FixedArrayParts(value); ok {
		stable, err := stableStateBodyType(element, coordinates)
		if err != nil {
			return "", err
		}
		return stable + "[" + strconv.FormatUint(length, 10) + "]", nil
	}
	if element, ok := hir.OptionElement(value); ok {
		stable, err := stableStateBodyType(element, coordinates)
		if err != nil {
			return "", err
		}
		return "Option<" + stable + ">", nil
	}
	if success, failure, ok := hir.ResultElements(value); ok {
		stableSuccess, err := stableStateBodyType(success, coordinates)
		if err != nil {
			return "", err
		}
		stableFailure, err := stableStateBodyType(failure, coordinates)
		if err != nil {
			return "", err
		}
		return "Result<" + stableSuccess + ", " + stableFailure + ">", nil
	}
	packageID, logicalName, ok := stateBodyNominalTypeParts(value)
	if !ok || !validStateBodyLogicalName(logicalName) {
		return "", fmt.Errorf("unsupported canonical type %q", value)
	}
	coordinate, found := coordinates[packageID]
	if !found {
		return "", fmt.Errorf("type %q references package %d without PackageCoordinateId", value, packageID)
	}
	return "pkg[" + coordinate.String() + "]::" + logicalName, nil
}

func validStateBodyLogicalName(value string) bool {
	segments := strings.Split(value, "::")
	if len(segments) < 2 {
		return false
	}
	for _, segment := range segments {
		if !source.ValidIdentifier(segment) {
			return false
		}
	}
	return true
}

func stateBodyNominalTypeParts(value hir.Type) (hir.PackageID, string, bool) {
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

func stateBodyBuiltinType(value hir.Type) bool {
	return value == hir.TypeVoid || value == hir.TypeBool || value == hir.TypeU8 || value == hir.TypeI32 ||
		value == hir.TypeU32 || value == hir.TypeUSize || value == hir.TypeByteSlice || value == hir.TypeStr ||
		value == hir.TypeCStr || value == hir.TypeRegion || hir.IsCScalar(value)
}

func findHIRFunctionOwner(input hir.Package, packageID hir.PackageID, symbol hir.SymbolID) (hir.Module, hir.Function, error) {
	var owner hir.Module
	var result hir.Function
	found := false
	for _, module := range input.Modules {
		if module.Package != packageID {
			continue
		}
		for _, function := range module.Functions {
			if function.Symbol != symbol {
				continue
			}
			if found {
				return hir.Module{}, hir.Function{}, fmt.Errorf("package %d function symbol %d is duplicated", packageID, symbol)
			}
			owner, result, found = module, function, true
		}
	}
	if !found {
		return hir.Module{}, hir.Function{}, fmt.Errorf("package %d state transaction names missing function %d", packageID, symbol)
	}
	return owner, result, nil
}

type stateBodyWriter interface {
	Write([]byte) (int, error)
}

func writeStateBodyValue(writer stateBodyWriter, value Value) {
	writeStateBodyString(writer, value.TypeID)
	writeStateBodyUint64(writer, uint64(len(value.Encoded)))
	_, _ = writer.Write(value.Encoded)
}

func writeStateBodyStrings(writer stateBodyWriter, values []hir.Effect) {
	writeStateBodyUint64(writer, uint64(len(values)))
	for _, value := range values {
		writeStateBodyString(writer, string(value))
	}
}

func writeStateBodyString(writer stateBodyWriter, value string) {
	writeStateBodyUint64(writer, uint64(len(value)))
	_, _ = writer.Write([]byte(value))
}

func writeStateBodyUint64(writer stateBodyWriter, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	_, _ = writer.Write(raw[:])
}
