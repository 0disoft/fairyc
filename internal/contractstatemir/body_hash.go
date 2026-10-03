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

type describedStateBody struct {
	coordinate   packagegraph.CoordinateID
	layout       Layout
	transactions []stableStateTransaction
}

// ComputeHIRStateBodyHashes verifies every transaction once, removes all
// build-local identities, and hashes each requested package independently.
// packageIDs may be in any order but must be unique.
func ComputeHIRStateBodyHashes(input hir.Package, packageIDs []hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.PackageID]StateBodyHash, error) {
	described, err := describeHIRStateBodyParts(input, packageIDs, coordinates)
	if err != nil {
		return nil, err
	}
	result := make(map[hir.PackageID]StateBodyHash, len(described))
	for packageID, body := range described {
		result[packageID] = hashStableStateBody(body.coordinate, body.layout, body.transactions)
	}
	return result, nil
}

func describeHIRStateBodyParts(input hir.Package, packageIDs []hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.PackageID]describedStateBody, error) {
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

	result := make(map[hir.PackageID]describedStateBody, len(requested))
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
		result[packageID] = describedStateBody{coordinate: coordinate, layout: layoutByPackage[packageID], transactions: items}
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
		if programHasBorrows(transaction.program) {
			writeStateBodyString(hasher, "borrows-v1")
			writeStateBodyBorrowExecutable(hasher, transaction.program.Effects, transaction.program.Operations, transaction.program.Exit, transaction.program.TrapCategory, transaction.program.Entry, transaction.program.Blocks)
			writeStateBodyUint64(hasher, uint64(len(transaction.program.Helpers)))
			for _, helper := range transaction.program.Helpers {
				writeStateBodyString(hasher, helper.Name)
				writeStateBodyBorrowExecutable(hasher, helper.Effects, helper.Operations, helper.Exit, helper.TrapCategory, helper.Entry, helper.Blocks)
			}
			continue
		}
		if len(transaction.program.Helpers) != 0 {
			writeStateBodyString(hasher, "helpers-v1")
			writeStateBodyExecutable(hasher, transaction.program.Effects, transaction.program.Operations, transaction.program.Exit, transaction.program.TrapCategory, transaction.program.Entry, transaction.program.Blocks)
			writeStateBodyUint64(hasher, uint64(len(transaction.program.Helpers)))
			for _, helper := range transaction.program.Helpers {
				writeStateBodyString(hasher, helper.Name)
				writeStateBodyExecutable(hasher, helper.Effects, helper.Operations, helper.Exit, helper.TrapCategory, helper.Entry, helper.Blocks)
			}
			continue
		}
		writeStateBodyStrings(hasher, transaction.program.Effects)
		if len(transaction.program.Blocks) == 0 {
			writeStateBodyUint64(hasher, uint64(len(transaction.program.Operations)))
			for _, operation := range transaction.program.Operations {
				writeStateBodyString(hasher, string(operation.Kind))
				writeStateBodyUint64(hasher, uint64(operation.FieldIndex))
				writeStateBodyValue(hasher, operation.Key)
				writeStateBodyValue(hasher, operation.Value)
			}
			writeStateBodyString(hasher, string(transaction.program.Exit))
			writeStateBodyString(hasher, transaction.program.TrapCategory)
		} else {
			writeStateBodyString(hasher, "graph-v1")
			writeStateBodyUint64(hasher, uint64(transaction.program.Entry))
			writeStateBodyUint64(hasher, uint64(len(transaction.program.Blocks)))
			for _, block := range transaction.program.Blocks {
				writeStateBodyUint64(hasher, uint64(block.ID))
				writeStateBodyUint64(hasher, uint64(len(block.Parameters)))
				for _, parameter := range block.Parameters {
					writeStateBodyUint64(hasher, uint64(parameter.ID))
					writeStateBodyString(hasher, parameter.TypeID)
				}
				writeStateBodyUint64(hasher, uint64(len(block.Operations)))
				for _, operation := range block.Operations {
					writeStateBodyOperation(hasher, operation)
				}
				writeStateBodyTerminator(hasher, block.Terminator)
			}
		}
	}
	var result StateBodyHash
	copy(result[:], hasher.Sum(nil))
	return result
}

func programHasBorrows(program Program) bool {
	if operationsHaveBorrows(program.Operations) || blocksHaveBorrows(program.Blocks) {
		return true
	}
	for _, helper := range program.Helpers {
		if operationsHaveBorrows(helper.Operations) || blocksHaveBorrows(helper.Blocks) {
			return true
		}
	}
	return false
}

func blocksHaveBorrows(blocks []Block) bool {
	for _, block := range blocks {
		if operationsHaveBorrows(block.Operations) {
			return true
		}
	}
	return false
}

func operationsHaveBorrows(operations []Operation) bool {
	for _, operation := range operations {
		if isBorrowOperation(operation.Kind) || operation.BorrowInput != 0 || operation.BorrowResult != 0 || operation.ViewOffset != 0 || operation.ViewLength != 0 {
			return true
		}
	}
	return false
}

func writeStateBodyBorrowExecutable(writer stateBodyWriter, effects []hir.Effect, operations []Operation, exit ExitKind, trapCategory string, entry BlockID, blocks []Block) {
	writeStateBodyStrings(writer, effects)
	if len(blocks) == 0 {
		writeStateBodyString(writer, "linear")
		writeStateBodyUint64(writer, uint64(len(operations)))
		for _, operation := range operations {
			writeStateBodyBorrowOperation(writer, operation)
		}
		writeStateBodyString(writer, string(exit))
		writeStateBodyString(writer, trapCategory)
		return
	}
	writeStateBodyString(writer, "graph")
	writeStateBodyUint64(writer, uint64(entry))
	writeStateBodyUint64(writer, uint64(len(blocks)))
	for _, block := range blocks {
		writeStateBodyUint64(writer, uint64(block.ID))
		writeStateBodyUint64(writer, uint64(len(block.Parameters)))
		for _, parameter := range block.Parameters {
			writeStateBodyUint64(writer, uint64(parameter.ID))
			writeStateBodyString(writer, parameter.TypeID)
		}
		writeStateBodyUint64(writer, uint64(len(block.Operations)))
		for _, operation := range block.Operations {
			writeStateBodyBorrowOperation(writer, operation)
		}
		writeStateBodyTerminator(writer, block.Terminator)
	}
}

func writeStateBodyBorrowOperation(writer stateBodyWriter, operation Operation) {
	writeStateBodyHelperOperation(writer, operation)
	writeStateBodyUint64(writer, uint64(operation.BorrowInput))
	writeStateBodyUint64(writer, uint64(operation.BorrowResult))
	writeStateBodyUint64(writer, uint64(operation.ViewOffset))
	writeStateBodyUint64(writer, uint64(operation.ViewLength))
}

func writeStateBodyExecutable(writer stateBodyWriter, effects []hir.Effect, operations []Operation, exit ExitKind, trapCategory string, entry BlockID, blocks []Block) {
	writeStateBodyStrings(writer, effects)
	if len(blocks) == 0 {
		writeStateBodyString(writer, "linear")
		writeStateBodyUint64(writer, uint64(len(operations)))
		for _, operation := range operations {
			writeStateBodyHelperOperation(writer, operation)
		}
		writeStateBodyString(writer, string(exit))
		writeStateBodyString(writer, trapCategory)
		return
	}
	writeStateBodyString(writer, "graph")
	writeStateBodyUint64(writer, uint64(entry))
	writeStateBodyUint64(writer, uint64(len(blocks)))
	for _, block := range blocks {
		writeStateBodyUint64(writer, uint64(block.ID))
		writeStateBodyUint64(writer, uint64(len(block.Parameters)))
		for _, parameter := range block.Parameters {
			writeStateBodyUint64(writer, uint64(parameter.ID))
			writeStateBodyString(writer, parameter.TypeID)
		}
		writeStateBodyUint64(writer, uint64(len(block.Operations)))
		for _, operation := range block.Operations {
			writeStateBodyHelperOperation(writer, operation)
		}
		writeStateBodyTerminator(writer, block.Terminator)
	}
}

func writeStateBodyHelperOperation(writer stateBodyWriter, operation Operation) {
	writeStateBodyOperation(writer, operation)
	writeStateBodyString(writer, operation.Callee)
}

func writeStateBodyOperation(writer stateBodyWriter, operation Operation) {
	writeStateBodyString(writer, string(operation.Kind))
	writeStateBodyUint64(writer, uint64(operation.FieldIndex))
	writeStateBodyValue(writer, operation.Key)
	writeStateBodyValue(writer, operation.Value)
	writeStateBodyUint64(writer, uint64(operation.KeyInput))
	writeStateBodyUint64(writer, uint64(operation.ValueInput))
	writeStateBodyUint64(writer, uint64(operation.Result))
	writeStateBodyUint64(writer, uint64(operation.Found))
	writeStateBodyUint64(writer, uint64(operation.Cost))
}

func writeStateBodyValueIDs(writer stateBodyWriter, values []ValueID) {
	writeStateBodyUint64(writer, uint64(len(values)))
	for _, value := range values {
		writeStateBodyUint64(writer, uint64(value))
	}
}

func writeStateBodyTerminator(writer stateBodyWriter, value Terminator) {
	writeStateBodyString(writer, string(value.Kind))
	writeStateBodyUint64(writer, uint64(value.Condition))
	writeStateBodyUint64(writer, uint64(value.Target))
	writeStateBodyValueIDs(writer, value.Arguments)
	writeStateBodyUint64(writer, uint64(value.TrueTarget))
	writeStateBodyValueIDs(writer, value.TrueArguments)
	writeStateBodyUint64(writer, uint64(value.FalseTarget))
	writeStateBodyValueIDs(writer, value.FalseArguments)
	writeStateBodyString(writer, value.TrapCategory)
	writeStateBodyUint64(writer, uint64(value.Cost))
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
	descriptor := stableStateFunctionDescriptor{
		name:           logicalName,
		parameterTypes: make([]string, 0, len(function.Parameters)),
		effects:        make([]string, 0, len(function.Effects)),
	}
	for _, parameter := range function.Parameters {
		stable, err := stableStateBodyType(parameter.Type, coordinates)
		if err != nil {
			return nil, fmt.Errorf("state function %q parameter type: %w", logicalName, err)
		}
		descriptor.parameterTypes = append(descriptor.parameterTypes, stable)
	}
	result, err := stableStateBodyType(function.ResultType, coordinates)
	if err != nil {
		return nil, fmt.Errorf("state function %q result type: %w", logicalName, err)
	}
	descriptor.resultType = result
	for _, effect := range function.Effects {
		descriptor.effects = append(descriptor.effects, string(effect))
	}
	if function.DropGlueFor != "" {
		stable, err := stableStateBodyType(function.DropGlueFor, coordinates)
		if err != nil {
			return nil, fmt.Errorf("state function %q drop-glue type: %w", logicalName, err)
		}
		descriptor.dropGlueFor = stable
	}
	return encodeStableStateFunctionDescriptor(descriptor), nil
}

func stableStateBodyType(value hir.Type, coordinates map[hir.PackageID]packagegraph.CoordinateID) (string, error) {
	if element, nominal := hir.StructSliceElement(value); nominal {
		stable, err := stableStateBodyType(element, coordinates)
		if err != nil {
			return "", err
		}
		return stable + "[]", nil
	}
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
	_, integerSlice := hir.IntegerViewElement(value)
	return value == hir.TypeVoid || value == hir.TypeBool || value == hir.TypeU8 || value == hir.TypeI32 ||
		value == hir.TypeU32 || value == hir.TypeUSize || integerSlice || value == hir.TypeStr ||
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
