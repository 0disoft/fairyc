package contractstatemir

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const (
	StableStateBodySchema        = "fairyc.contract-state-package-body-artifact/v1"
	StableStateBodyVersion       = "fairyc-contract-state-package-body-artifact/1"
	StableStateGraphBodySchema   = "fairyc.contract-state-package-body-artifact/v2"
	StableStateGraphBodyVersion  = "fairyc-contract-state-package-body-artifact/2"
	StableStateHelperBodySchema  = "fairyc.contract-state-package-body-artifact/v3"
	StableStateHelperBodyVersion = "fairyc-contract-state-package-body-artifact/3"
)

var (
	ErrStableStateBodyTooLarge = errors.New("stable contract-state body artifact exceeds the caller byte limit")
	ErrStableStateBodyEncoding = errors.New("stable contract-state body artifact is not canonically encoded")
)

// StableStateBodyArtifact is one package's exact transaction payload without
// build-local package, function, span, or capability identities.
type StableStateBodyArtifact struct {
	Package      packagegraph.CoordinateID `json:"package"`
	Hash         StateBodyHash             `json:"hash"`
	Layout       *StableStateLayout        `json:"layout"`
	Transactions []StableStateTransaction  `json:"transactions"`
}

type StableStateLayout struct {
	ContractUID string             `json:"contractUid"`
	Fields      []StableStateField `json:"fields"`
}

type StableStateField struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	KeyTypeID   string `json:"keyTypeId"`
	ValueTypeID string `json:"valueTypeId"`
}

type StableStateTransaction struct {
	Function     []byte                 `json:"function"`
	Effects      []string               `json:"effects"`
	Operations   []StableStateOperation `json:"operations"`
	Exit         string                 `json:"exit"`
	TrapCategory string                 `json:"trapCategory"`
	Entry        uint32                 `json:"entry,omitempty"`
	Blocks       []StableStateBlock     `json:"blocks,omitempty"`
	Helpers      []StableStateHelper    `json:"helpers,omitempty"`
}

type StableStateHelper struct {
	Name         string                 `json:"name"`
	Effects      []string               `json:"effects"`
	Operations   []StableStateOperation `json:"operations"`
	Exit         string                 `json:"exit"`
	TrapCategory string                 `json:"trapCategory"`
	Entry        uint32                 `json:"entry,omitempty"`
	Blocks       []StableStateBlock     `json:"blocks,omitempty"`
}

type StableStateOperation struct {
	Kind       string           `json:"kind"`
	FieldIndex uint32           `json:"fieldIndex"`
	Key        StableStateValue `json:"key"`
	Value      StableStateValue `json:"value"`
	KeyInput   uint32           `json:"keyInput,omitempty"`
	ValueInput uint32           `json:"valueInput,omitempty"`
	Result     uint32           `json:"result,omitempty"`
	Found      uint32           `json:"found,omitempty"`
	Callee     string           `json:"callee,omitempty"`
	Cost       uint32           `json:"cost,omitempty"`
}

type StableStateBlock struct {
	ID         uint32                      `json:"id"`
	Parameters []StableStateBlockParameter `json:"parameters"`
	Operations []StableStateOperation      `json:"operations"`
	Terminator StableStateTerminator       `json:"terminator"`
}

type StableStateBlockParameter struct {
	ID     uint32 `json:"id"`
	TypeID string `json:"typeId"`
}

type StableStateTerminator struct {
	Kind           string   `json:"kind"`
	Condition      uint32   `json:"condition"`
	Target         uint32   `json:"target"`
	Arguments      []uint32 `json:"arguments"`
	TrueTarget     uint32   `json:"trueTarget"`
	TrueArguments  []uint32 `json:"trueArguments"`
	FalseTarget    uint32   `json:"falseTarget"`
	FalseArguments []uint32 `json:"falseArguments"`
	TrapCategory   string   `json:"trapCategory"`
	Cost           uint32   `json:"cost"`
}

type StableStateValue struct {
	TypeID  string `json:"typeId"`
	Encoded []byte `json:"encoded"`
}

type stableStateBodyEnvelope struct {
	Schema  string                  `json:"schema"`
	Version string                  `json:"version"`
	Body    StableStateBodyArtifact `json:"body"`
}

type stableStateFunctionDescriptor struct {
	name           string
	parameterTypes []string
	resultType     string
	effects        []string
	dropGlueFor    string
}

// DescribeStableHIRStateBodies verifies and projects each requested package
// into a deterministic capability-free payload artifact.
func DescribeStableHIRStateBodies(input hir.Package, packageIDs []hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.PackageID]StableStateBodyArtifact, error) {
	described, err := describeHIRStateBodyParts(input, packageIDs, coordinates)
	if err != nil {
		return nil, err
	}
	result := make(map[hir.PackageID]StableStateBodyArtifact, len(described))
	for packageID, body := range described {
		artifact := StableStateBodyArtifact{
			Package: body.coordinate, Layout: stableStateLayout(body.layout),
			Transactions: make([]StableStateTransaction, 0, len(body.transactions)),
		}
		for _, transaction := range body.transactions {
			artifact.Transactions = append(artifact.Transactions, stableStateTransactionArtifact(transaction))
		}
		artifact.Hash = hashStableStateBody(body.coordinate, body.layout, body.transactions)
		if err := ValidateStableStateBody(artifact); err != nil {
			return nil, fmt.Errorf("validate package %d stable contract-state body: %w", packageID, err)
		}
		result[packageID] = artifact
	}
	return result, nil
}

// EncodeStableStateBody returns canonical exact-version JSON under a mandatory
// caller byte limit.
func EncodeStableStateBody(body StableStateBodyArtifact, maxBytes uint64) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("stable contract-state body byte limit must be positive")
	}
	if err := ValidateStableStateBody(body); err != nil {
		return nil, err
	}
	schema, version := stableStateBodyEnvelopeIdentity(body)
	encoded, err := json.Marshal(stableStateBodyEnvelope{Schema: schema, Version: version, Body: body})
	if err != nil {
		return nil, fmt.Errorf("encode stable contract-state body artifact: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrStableStateBodyTooLarge
	}
	return encoded, nil
}

// DecodeStableStateBody accepts one strict canonical exact-version JSON value
// and independently validates every payload invariant before returning it.
func DecodeStableStateBody(encoded []byte, maxBytes uint64) (StableStateBodyArtifact, error) {
	if maxBytes == 0 {
		return StableStateBodyArtifact{}, fmt.Errorf("stable contract-state body byte limit must be positive")
	}
	if len(encoded) == 0 {
		return StableStateBodyArtifact{}, fmt.Errorf("stable contract-state body artifact is empty")
	}
	if uint64(len(encoded)) > maxBytes {
		return StableStateBodyArtifact{}, ErrStableStateBodyTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope stableStateBodyEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return StableStateBodyArtifact{}, fmt.Errorf("decode stable contract-state body artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return StableStateBodyArtifact{}, fmt.Errorf("decode stable contract-state body artifact: trailing JSON value")
		}
		return StableStateBodyArtifact{}, fmt.Errorf("decode stable contract-state body artifact trailing data: %w", err)
	}
	expectedSchema, expectedVersion := stableStateBodyEnvelopeIdentity(envelope.Body)
	if envelope.Schema != expectedSchema || envelope.Version != expectedVersion {
		return StableStateBodyArtifact{}, fmt.Errorf("stable contract-state body schema or version is incompatible")
	}
	if err := ValidateStableStateBody(envelope.Body); err != nil {
		return StableStateBodyArtifact{}, fmt.Errorf("verify decoded stable contract-state body artifact: %w", err)
	}
	canonical, err := EncodeStableStateBody(envelope.Body, maxBytes)
	if err != nil {
		return StableStateBodyArtifact{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return StableStateBodyArtifact{}, ErrStableStateBodyEncoding
	}
	return envelope.Body, nil
}

func stableStateBodyEnvelopeIdentity(body StableStateBodyArtifact) (string, string) {
	for _, transaction := range body.Transactions {
		if len(transaction.Helpers) != 0 {
			return StableStateHelperBodySchema, StableStateHelperBodyVersion
		}
		if len(transaction.Blocks) != 0 {
			return StableStateGraphBodySchema, StableStateGraphBodyVersion
		}
	}
	return StableStateBodySchema, StableStateBodyVersion
}

// ValidateStableStateBody reconstructs the exact layout and verifier-ready
// transaction programs without trusting serialized capability authority.
func ValidateStableStateBody(body StableStateBodyArtifact) error {
	if body.Package == (packagegraph.CoordinateID{}) || body.Hash == (StateBodyHash{}) {
		return fmt.Errorf("stable contract-state body identity is incomplete")
	}
	if body.Transactions == nil {
		return fmt.Errorf("stable contract-state body transactions must be an array")
	}
	layout, err := layoutFromStableState(body.Layout)
	if err != nil {
		return err
	}
	transactions := make([]stableStateTransaction, 0, len(body.Transactions))
	var previous []byte
	for index, transaction := range body.Transactions {
		descriptor, err := decodeStableStateFunctionDescriptor(transaction.Function)
		if err != nil {
			return fmt.Errorf("stable transaction %d function: %w", index, err)
		}
		if index > 0 && bytes.Compare(previous, transaction.Function) >= 0 {
			return fmt.Errorf("stable contract-state transactions are not sorted and unique by function")
		}
		previous = transaction.Function
		program, err := programFromStableStateTransaction(layout, descriptor, transaction)
		if err != nil {
			return fmt.Errorf("stable transaction %d: %w", index, err)
		}
		transactions = append(transactions, stableStateTransaction{function: slices.Clone(transaction.Function), program: program})
	}
	computed := hashStableStateBody(body.Package, layout, transactions)
	if computed != body.Hash {
		return fmt.Errorf("stable contract-state body hash does not match its payload")
	}
	return nil
}

func stableStateLayout(layout Layout) *StableStateLayout {
	if layout.Hash == (contractstate.LayoutHash{}) {
		return nil
	}
	result := &StableStateLayout{ContractUID: layout.ContractUID, Fields: make([]StableStateField, 0, len(layout.Fields))}
	for _, field := range layout.Fields {
		result.Fields = append(result.Fields, StableStateField{
			Name: field.Name, Kind: field.Kind.String(), KeyTypeID: field.KeyTypeID, ValueTypeID: field.ValueTypeID,
		})
	}
	return result
}

func stableStateTransactionArtifact(transaction stableStateTransaction) StableStateTransaction {
	result := StableStateTransaction{
		Function: slices.Clone(transaction.function), Effects: make([]string, 0, len(transaction.program.Effects)),
		Operations: make([]StableStateOperation, 0, len(transaction.program.Operations)),
		Exit:       string(transaction.program.Exit), TrapCategory: transaction.program.TrapCategory,
		Entry: uint32(transaction.program.Entry),
	}
	for _, effect := range transaction.program.Effects {
		result.Effects = append(result.Effects, string(effect))
	}
	for _, operation := range transaction.program.Operations {
		result.Operations = append(result.Operations, stableStateOperation(operation))
	}
	if len(transaction.program.Blocks) != 0 {
		result.Blocks = make([]StableStateBlock, 0, len(transaction.program.Blocks))
	}
	for _, block := range transaction.program.Blocks {
		stableBlock := StableStateBlock{
			ID: uint32(block.ID), Parameters: make([]StableStateBlockParameter, 0, len(block.Parameters)),
			Operations: make([]StableStateOperation, 0, len(block.Operations)),
			Terminator: stableStateTerminator(block.Terminator),
		}
		for _, parameter := range block.Parameters {
			stableBlock.Parameters = append(stableBlock.Parameters, StableStateBlockParameter{ID: uint32(parameter.ID), TypeID: parameter.TypeID})
		}
		for _, operation := range block.Operations {
			stableBlock.Operations = append(stableBlock.Operations, stableStateOperation(operation))
		}
		result.Blocks = append(result.Blocks, stableBlock)
	}
	if len(transaction.program.Helpers) != 0 {
		result.Helpers = make([]StableStateHelper, 0, len(transaction.program.Helpers))
	}
	for _, helper := range transaction.program.Helpers {
		stableHelper := StableStateHelper{
			Name: helper.Name, Effects: stableEffects(helper.Effects), Operations: stableOperations(helper.Operations),
			Exit: string(helper.Exit), TrapCategory: helper.TrapCategory, Entry: uint32(helper.Entry), Blocks: stableBlocks(helper.Blocks),
		}
		result.Helpers = append(result.Helpers, stableHelper)
	}
	return result
}

func stableEffects(effects []hir.Effect) []string {
	result := make([]string, 0, len(effects))
	for _, effect := range effects {
		result = append(result, string(effect))
	}
	return result
}

func stableOperations(operations []Operation) []StableStateOperation {
	result := make([]StableStateOperation, 0, len(operations))
	for _, operation := range operations {
		result = append(result, stableStateOperation(operation))
	}
	return result
}

func stableBlocks(blocks []Block) []StableStateBlock {
	if blocks == nil {
		return nil
	}
	result := make([]StableStateBlock, 0, len(blocks))
	for _, block := range blocks {
		stableBlock := StableStateBlock{ID: uint32(block.ID), Parameters: make([]StableStateBlockParameter, 0, len(block.Parameters)), Operations: stableOperations(block.Operations), Terminator: stableStateTerminator(block.Terminator)}
		for _, parameter := range block.Parameters {
			stableBlock.Parameters = append(stableBlock.Parameters, StableStateBlockParameter{ID: uint32(parameter.ID), TypeID: parameter.TypeID})
		}
		result = append(result, stableBlock)
	}
	return result
}

func stableStateOperation(operation Operation) StableStateOperation {
	return StableStateOperation{
		Kind: string(operation.Kind), FieldIndex: operation.FieldIndex,
		Key: stableStateValue(operation.Key), Value: stableStateValue(operation.Value),
		KeyInput: uint32(operation.KeyInput), ValueInput: uint32(operation.ValueInput),
		Result: uint32(operation.Result), Found: uint32(operation.Found), Cost: operation.Cost,
		Callee: operation.Callee,
	}
}

func stableStateTerminator(value Terminator) StableStateTerminator {
	return StableStateTerminator{
		Kind: string(value.Kind), Condition: uint32(value.Condition), Target: uint32(value.Target),
		Arguments: stableStateValueIDs(value.Arguments), TrueTarget: uint32(value.TrueTarget),
		TrueArguments: stableStateValueIDs(value.TrueArguments), FalseTarget: uint32(value.FalseTarget),
		FalseArguments: stableStateValueIDs(value.FalseArguments), TrapCategory: value.TrapCategory, Cost: value.Cost,
	}
}

func stableStateValueIDs(values []ValueID) []uint32 {
	result := make([]uint32, 0, len(values))
	for _, value := range values {
		result = append(result, uint32(value))
	}
	return result
}

func stableStateValue(value Value) StableStateValue {
	return StableStateValue{TypeID: value.TypeID, Encoded: slices.Clone(value.Encoded)}
}

func layoutFromStableState(value *StableStateLayout) (Layout, error) {
	if value == nil {
		return Layout{}, nil
	}
	if value.Fields == nil {
		return Layout{}, fmt.Errorf("stable contract-state layout fields must be an array")
	}
	declarations := make([]Declaration, 0, len(value.Fields))
	for index, field := range value.Fields {
		var kind contractstate.FieldKind
		switch field.Kind {
		case "cell":
			kind = contractstate.FieldCell
		case "map":
			kind = contractstate.FieldMap
		default:
			return Layout{}, fmt.Errorf("stable contract-state layout field %d has unsupported kind %q", index, field.Kind)
		}
		declarations = append(declarations, Declaration{
			Name: field.Name, Kind: kind, KeyTypeID: field.KeyTypeID, ValueTypeID: field.ValueTypeID,
		})
	}
	layout, err := BuildLayout(value.ContractUID, declarations)
	if err != nil {
		return Layout{}, fmt.Errorf("rebuild stable contract-state layout: %w", err)
	}
	return layout, nil
}

func programFromStableStateTransaction(layout Layout, descriptor stableStateFunctionDescriptor, value StableStateTransaction) (Program, error) {
	if value.Effects == nil || value.Operations == nil {
		return Program{}, fmt.Errorf("stable transaction effects and operations must be arrays")
	}
	if !slices.Equal(descriptor.effects, value.Effects) {
		return Program{}, fmt.Errorf("stable transaction effects differ from its function descriptor")
	}
	effects := make([]hir.Effect, 0, len(value.Effects))
	for _, effect := range value.Effects {
		parsed, ok := hir.ParseEffect(effect)
		if !ok {
			return Program{}, fmt.Errorf("stable transaction has unknown effect %q", effect)
		}
		effects = append(effects, parsed)
	}
	capability := newTransactionCapability()
	operations := make([]Operation, 0, len(value.Operations))
	for _, operation := range value.Operations {
		operations = append(operations, operationFromStable(operation, capability))
	}
	blocks := make([]Block, 0, len(value.Blocks))
	for _, stableBlock := range value.Blocks {
		block := Block{
			ID: BlockID(stableBlock.ID), Parameters: make([]BlockParameter, 0, len(stableBlock.Parameters)),
			Operations: make([]Operation, 0, len(stableBlock.Operations)),
			Terminator: terminatorFromStable(stableBlock.Terminator),
		}
		for _, parameter := range stableBlock.Parameters {
			block.Parameters = append(block.Parameters, BlockParameter{ID: ValueID(parameter.ID), TypeID: parameter.TypeID})
		}
		for _, operation := range stableBlock.Operations {
			block.Operations = append(block.Operations, operationFromStable(operation, capability))
		}
		blocks = append(blocks, block)
	}
	helpers := make([]Helper, 0, len(value.Helpers))
	for index, stableHelper := range value.Helpers {
		if stableHelper.Operations == nil {
			return Program{}, fmt.Errorf("stable helper %d operations must be an array", index)
		}
		helperEffects, err := effectsFromStable(stableHelper.Effects)
		if err != nil {
			return Program{}, fmt.Errorf("stable helper %d: %w", index, err)
		}
		helpers = append(helpers, Helper{
			Name: stableHelper.Name, Effects: helperEffects,
			Operations: operationsFromStable(stableHelper.Operations, capability),
			Exit:       ExitKind(stableHelper.Exit), TrapCategory: stableHelper.TrapCategory,
			Entry: BlockID(stableHelper.Entry), Blocks: blocksFromStable(stableHelper.Blocks, capability),
		})
	}
	program := Program{
		Layout: layout, Capability: capability, Effects: effects, Operations: operations,
		Exit: ExitKind(value.Exit), TrapCategory: value.TrapCategory, Entry: BlockID(value.Entry), Blocks: blocks,
		Helpers: helpers,
	}
	if failures := Verify(program); len(failures) != 0 {
		return Program{}, VerificationErrors{Failures: failures}
	}
	return program, nil
}

func operationFromStable(value StableStateOperation, capability TransactionCapability) Operation {
	return Operation{
		Kind: OperationKind(value.Kind), Capability: capability, FieldIndex: value.FieldIndex,
		Key:      Value{TypeID: value.Key.TypeID, Encoded: slices.Clone(value.Key.Encoded)},
		Value:    Value{TypeID: value.Value.TypeID, Encoded: slices.Clone(value.Value.Encoded)},
		KeyInput: ValueID(value.KeyInput), ValueInput: ValueID(value.ValueInput),
		Result: ValueID(value.Result), Found: ValueID(value.Found), Cost: value.Cost,
		Callee: value.Callee,
	}
}

func effectsFromStable(values []string) ([]hir.Effect, error) {
	if values == nil {
		return nil, fmt.Errorf("effects must be an array")
	}
	result := make([]hir.Effect, 0, len(values))
	for _, value := range values {
		parsed, ok := hir.ParseEffect(value)
		if !ok {
			return nil, fmt.Errorf("unknown effect %q", value)
		}
		result = append(result, parsed)
	}
	return result, nil
}

func operationsFromStable(values []StableStateOperation, capability TransactionCapability) []Operation {
	result := make([]Operation, 0, len(values))
	for _, value := range values {
		result = append(result, operationFromStable(value, capability))
	}
	return result
}

func blocksFromStable(values []StableStateBlock, capability TransactionCapability) []Block {
	if values == nil {
		return nil
	}
	result := make([]Block, 0, len(values))
	for _, stableBlock := range values {
		block := Block{ID: BlockID(stableBlock.ID), Parameters: make([]BlockParameter, 0, len(stableBlock.Parameters)), Operations: operationsFromStable(stableBlock.Operations, capability), Terminator: terminatorFromStable(stableBlock.Terminator)}
		for _, parameter := range stableBlock.Parameters {
			block.Parameters = append(block.Parameters, BlockParameter{ID: ValueID(parameter.ID), TypeID: parameter.TypeID})
		}
		result = append(result, block)
	}
	return result
}

func terminatorFromStable(value StableStateTerminator) Terminator {
	return Terminator{
		Kind: TerminatorKind(value.Kind), Condition: ValueID(value.Condition), Target: BlockID(value.Target),
		Arguments: valueIDsFromStable(value.Arguments), TrueTarget: BlockID(value.TrueTarget),
		TrueArguments: valueIDsFromStable(value.TrueArguments), FalseTarget: BlockID(value.FalseTarget),
		FalseArguments: valueIDsFromStable(value.FalseArguments), TrapCategory: value.TrapCategory, Cost: value.Cost,
	}
}

func valueIDsFromStable(values []uint32) []ValueID {
	result := make([]ValueID, 0, len(values))
	for _, value := range values {
		result = append(result, ValueID(value))
	}
	return result
}

func decodeStableStateFunctionDescriptor(encoded []byte) (stableStateFunctionDescriptor, error) {
	if len(encoded) == 0 {
		return stableStateFunctionDescriptor{}, fmt.Errorf("stable function descriptor is empty")
	}
	reader := bytes.NewReader(encoded)
	name, err := readStableStateString(reader)
	if err != nil {
		return stableStateFunctionDescriptor{}, err
	}
	parameterTypes, err := readStableStateStrings(reader)
	if err != nil {
		return stableStateFunctionDescriptor{}, err
	}
	resultType, err := readStableStateString(reader)
	if err != nil {
		return stableStateFunctionDescriptor{}, err
	}
	effects, err := readStableStateStrings(reader)
	if err != nil {
		return stableStateFunctionDescriptor{}, err
	}
	dropGlueFor, err := readStableStateString(reader)
	if err != nil {
		return stableStateFunctionDescriptor{}, err
	}
	if reader.Len() != 0 {
		return stableStateFunctionDescriptor{}, fmt.Errorf("stable function descriptor has trailing bytes")
	}
	descriptor := stableStateFunctionDescriptor{
		name: name, parameterTypes: parameterTypes, resultType: resultType, effects: effects, dropGlueFor: dropGlueFor,
	}
	if err := validateStableStateFunctionDescriptor(descriptor); err != nil {
		return stableStateFunctionDescriptor{}, err
	}
	canonical := encodeStableStateFunctionDescriptor(descriptor)
	if !bytes.Equal(encoded, canonical) {
		return stableStateFunctionDescriptor{}, fmt.Errorf("stable function descriptor is not canonical")
	}
	return descriptor, nil
}

func encodeStableStateFunctionDescriptor(value stableStateFunctionDescriptor) []byte {
	var encoded bytes.Buffer
	writeStateBodyString(&encoded, value.name)
	writeStableStateStrings(&encoded, value.parameterTypes)
	writeStateBodyString(&encoded, value.resultType)
	writeStableStateStrings(&encoded, value.effects)
	writeStateBodyString(&encoded, value.dropGlueFor)
	return encoded.Bytes()
}

func validateStableStateFunctionDescriptor(value stableStateFunctionDescriptor) error {
	if !validStateBodyLogicalName(value.name) {
		return fmt.Errorf("stable function descriptor has invalid logical name %q", value.name)
	}
	for _, parameterType := range value.parameterTypes {
		if !validStableStateBodyType(parameterType) {
			return fmt.Errorf("stable function descriptor has invalid parameter type %q", parameterType)
		}
	}
	if !validStableStateBodyType(value.resultType) {
		return fmt.Errorf("stable function descriptor has invalid result type %q", value.resultType)
	}
	if !slices.IsSorted(value.effects) {
		return fmt.Errorf("stable function descriptor effects are not sorted")
	}
	for index, effect := range value.effects {
		if _, ok := hir.ParseEffect(effect); !ok || index > 0 && value.effects[index-1] == effect {
			return fmt.Errorf("stable function descriptor effects are unknown or duplicated")
		}
	}
	if value.dropGlueFor != "" && !validStableStateBodyType(value.dropGlueFor) {
		return fmt.Errorf("stable function descriptor has invalid drop-glue type %q", value.dropGlueFor)
	}
	return nil
}

func validStableStateBodyType(value string) bool {
	canonical, ok := canonicalStableStateBodyType(value)
	return ok && canonical == value
}

func canonicalStableStateBodyType(value string) (string, bool) {
	if stateBodyBuiltinType(hir.Type(value)) {
		return value, true
	}
	for _, prefix := range []string{"ref mut ", "ref ", "ptr mut ", "ptr "} {
		if strings.HasPrefix(value, prefix) {
			element, ok := canonicalStableStateBodyType(strings.TrimPrefix(value, prefix))
			if !ok {
				return "", false
			}
			return prefix + element, true
		}
	}
	if element, ok := hir.OptionElement(hir.Type(value)); ok {
		canonical, valid := canonicalStableStateBodyType(string(element))
		if !valid {
			return "", false
		}
		return "Option<" + canonical + ">", true
	}
	if success, failure, ok := hir.ResultElements(hir.Type(value)); ok {
		canonicalSuccess, successValid := canonicalStableStateBodyType(string(success))
		canonicalFailure, failureValid := canonicalStableStateBodyType(string(failure))
		if !successValid || !failureValid {
			return "", false
		}
		return "Result<" + canonicalSuccess + ", " + canonicalFailure + ">", true
	}
	if strings.HasSuffix(value, "]") {
		separator := strings.LastIndexByte(value, '[')
		if separator > 0 {
			length, err := strconv.ParseUint(value[separator+1:len(value)-1], 10, 64)
			element, valid := canonicalStableStateBodyType(value[:separator])
			if err == nil && length > 0 && valid {
				return element + "[" + strconv.FormatUint(length, 10) + "]", true
			}
		}
	}
	const prefix = "pkg["
	if !strings.HasPrefix(value, prefix) {
		return "", false
	}
	separator := strings.Index(value, "]::")
	if separator <= len(prefix) || separator+3 >= len(value) {
		return "", false
	}
	coordinate, err := packagegraph.ParseCoordinateID(value[len(prefix):separator])
	if err != nil {
		return "", false
	}
	logicalName := value[separator+3:]
	if !validStateBodyLogicalName(logicalName) {
		return "", false
	}
	return "pkg[" + coordinate.String() + "]::" + logicalName, true
}

func readStableStateStrings(reader *bytes.Reader) ([]string, error) {
	count, err := readStableStateUint64(reader)
	if err != nil {
		return nil, err
	}
	if count > uint64(reader.Len()) || count > uint64(^uint(0)>>1) {
		return nil, fmt.Errorf("stable function descriptor string count exceeds its bytes")
	}
	result := make([]string, 0, int(count))
	for index := uint64(0); index < count; index++ {
		value, err := readStableStateString(reader)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func readStableStateString(reader *bytes.Reader) (string, error) {
	length, err := readStableStateUint64(reader)
	if err != nil {
		return "", err
	}
	if length > uint64(reader.Len()) || length > uint64(^uint(0)>>1) {
		return "", fmt.Errorf("stable function descriptor string exceeds its bytes")
	}
	value := make([]byte, int(length))
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", err
	}
	return string(value), nil
}

func readStableStateUint64(reader *bytes.Reader) (uint64, error) {
	var raw [8]byte
	if _, err := io.ReadFull(reader, raw[:]); err != nil {
		return 0, fmt.Errorf("read stable function descriptor: %w", err)
	}
	return binary.BigEndian.Uint64(raw[:]), nil
}

func writeStableStateStrings(writer stateBodyWriter, values []string) {
	writeStateBodyUint64(writer, uint64(len(values)))
	for _, value := range values {
		writeStateBodyString(writer, value)
	}
}
