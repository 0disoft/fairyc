package contractstatefixture

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

const (
	fixtureSchemaV0 = "fairyc.contract-state-fixtures/v0"
	fixtureSchemaV1 = "fairyc.contract-state-fixtures/v1"
	fixtureSchemaV2 = "fairyc.contract-state-fixtures/v2"
	fixtureSchemaV3 = "fairyc.contract-state-fixtures/v3"
)

var fixtureSchemaVersions = []string{fixtureSchemaV0, fixtureSchemaV1, fixtureSchemaV2, fixtureSchemaV3}

var operationKinds = []string{
	"cell-delete",
	"cell-read",
	"cell-write",
	"map-delete",
	"map-read",
	"map-write",
}

var helperOperationKinds = append(slices.Clone(operationKinds), "call")

var exitKinds = []string{"commit", "reject", "trap"}
var resultStatuses = []string{"commit", "error", "reject", "trap"}

// Run lowers and executes every manifest case against a fresh semantic-oracle store.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	if err := validateManifest(definition); err != nil {
		return Report{}, err
	}
	report := Report{CasesRun: len(definition.Cases) + len(definition.Mutations)}
	for _, fixture := range definition.Cases {
		if failure := runCase(definition, fixture); failure != nil {
			report.Failures = append(report.Failures, *failure)
		}
	}
	for _, mutation := range definition.Mutations {
		if failure := runMutation(definition, mutation); failure != nil {
			report.Failures = append(report.Failures, *failure)
		}
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var definition manifest
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode manifest: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	return definition, nil
}

func validateManifest(definition manifest) error {
	if definition.Schema == "" || !slices.Contains(fixtureSchemaVersions, definition.SchemaVersion) || len(definition.Contracts) == 0 {
		return fmt.Errorf("invalid contract-state fixture contract header")
	}
	if definition.Layout.ContractUID == "" || len(definition.Layout.Fields) == 0 {
		return fmt.Errorf("contract-state fixture layout is empty")
	}
	if err := validateLimits(definition.Limits); err != nil {
		return fmt.Errorf("contract-state fixture limits: %w", err)
	}
	fields := make(map[string]fieldDefinition, len(definition.Layout.Fields))
	for _, field := range definition.Layout.Fields {
		if field.Name == "" || field.ValueTypeID == "" {
			return fmt.Errorf("contract-state fixture field has an empty identity")
		}
		if _, duplicate := fields[field.Name]; duplicate {
			return fmt.Errorf("contract-state fixture field %q is duplicated", field.Name)
		}
		if field.Kind != "cell" && field.Kind != "map" {
			return fmt.Errorf("contract-state fixture field %q has unsupported kind %q", field.Name, field.Kind)
		}
		if (field.Kind == "cell") != (field.KeyTypeID == "") {
			return fmt.Errorf("contract-state fixture field %q has an invalid key type", field.Name)
		}
		fields[field.Name] = field
	}
	seen := make(map[string]struct{}, len(definition.Cases))
	cases := make(map[string]caseDefinition, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if fixture.ID == "" {
			return fmt.Errorf("contract-state fixture case has an empty id")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return fmt.Errorf("contract-state fixture case %q is duplicated", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		cases[fixture.ID] = fixture
		if definition.SchemaVersion != fixtureSchemaV3 && len(fixture.Helpers) != 0 {
			return fmt.Errorf("contract-state fixture case %q uses helpers before fixture schema v3", fixture.ID)
		}
		if definition.SchemaVersion == fixtureSchemaV3 && len(fixture.Helpers) == 0 {
			return fmt.Errorf("contract-state fixture case %q has no v3 helper", fixture.ID)
		}
		helperNames := make(map[string]struct{}, len(fixture.Helpers))
		for _, helper := range fixture.Helpers {
			if helper.Name == "" {
				return fmt.Errorf("contract-state fixture case %q has an unnamed helper", fixture.ID)
			}
			if _, duplicate := helperNames[helper.Name]; duplicate {
				return fmt.Errorf("contract-state fixture case %q duplicates helper %q", fixture.ID, helper.Name)
			}
			helperNames[helper.Name] = struct{}{}
			if len(helper.Blocks) != 0 || !slices.Contains([]string{"return", "reject", "trap"}, helper.Exit) {
				return fmt.Errorf("contract-state fixture case %q helper %q has an unsupported exit", fixture.ID, helper.Name)
			}
		}
		validExit := len(fixture.Blocks) == 0 && slices.Contains(exitKinds, fixture.Exit) || len(fixture.Blocks) != 0 && fixture.Exit == ""
		if !validExit || !slices.Contains(resultStatuses, fixture.Expect.Status) {
			return fmt.Errorf("contract-state fixture case %q has an unsupported exit or status", fixture.ID)
		}
		if fixture.Limits != nil {
			if err := validateLimits(*fixture.Limits); err != nil {
				return fmt.Errorf("contract-state fixture case %q limits: %w", fixture.ID, err)
			}
		}
		operations := slices.Clone(fixture.Operations)
		for _, block := range fixture.Blocks {
			operations = append(operations, block.Operations...)
		}
		for _, helper := range fixture.Helpers {
			operations = append(operations, helper.Operations...)
			for _, block := range helper.Blocks {
				operations = append(operations, block.Operations...)
			}
		}
		for _, operation := range operations {
			if operation.Kind == "call" {
				if definition.SchemaVersion != fixtureSchemaV3 {
					return fmt.Errorf("contract-state fixture case %q uses calls before fixture schema v3", fixture.ID)
				}
				if _, found := helperNames[operation.Callee]; !found || operation.Field != "" {
					return fmt.Errorf("contract-state fixture case %q calls invalid helper %q", fixture.ID, operation.Callee)
				}
				continue
			}
			field, found := fields[operation.Field]
			vocabulary := operationKinds
			if definition.SchemaVersion == fixtureSchemaV3 {
				vocabulary = helperOperationKinds
			}
			if !found || !slices.Contains(vocabulary, operation.Kind) || !strings.HasPrefix(operation.Kind, field.Kind+"-") {
				return fmt.Errorf("contract-state fixture case %q has an invalid operation on %q", fixture.ID, operation.Field)
			}
		}
	}
	for _, mutation := range definition.Mutations {
		if mutation.ID == "" || mutation.Case == "" || mutation.Kind == "" || mutation.ErrorContains == "" {
			return fmt.Errorf("contract-state mutation fixture has an empty identity")
		}
		if _, duplicate := seen[mutation.ID]; duplicate {
			return fmt.Errorf("contract-state fixture identity %q is duplicated", mutation.ID)
		}
		seen[mutation.ID] = struct{}{}
		if _, found := cases[mutation.Case]; !found {
			return fmt.Errorf("contract-state mutation %q names unknown case %q", mutation.ID, mutation.Case)
		}
		mutationKinds := []string{"cycle", "join-type", "missing-cost", "unknown-value"}
		if definition.SchemaVersion == fixtureSchemaV3 {
			mutationKinds = []string{"helper-cycle", "helper-effect"}
		}
		if !slices.Contains(mutationKinds, mutation.Kind) {
			return fmt.Errorf("contract-state mutation %q has unsupported kind %q", mutation.ID, mutation.Kind)
		}
	}
	return nil
}

func runMutation(definition manifest, mutation mutationDefinition) *Failure {
	var fixture caseDefinition
	for _, candidate := range definition.Cases {
		if candidate.ID == mutation.Case {
			fixture = candidate
			break
		}
	}
	input, err := buildHIRPackage(definition.Layout, fixture)
	if err != nil {
		return &Failure{CaseID: mutation.ID, Detail: err.Error()}
	}
	transaction := &input.StateTransactions[0]
	switch mutation.Kind {
	case "cycle":
		transaction.Blocks[len(transaction.Blocks)-1].Terminator = hir.StateTerminator{Kind: hir.StateTerminatorJump, Target: transaction.Entry, Arguments: []hir.StateValueID{}, Cost: 1}
	case "join-type":
		for index := range transaction.Blocks {
			if len(transaction.Blocks[index].Parameters) != 0 {
				transaction.Blocks[index].Parameters[0].TypeID = contractstate.Bool.TypeID()
				break
			}
		}
	case "missing-cost":
		transaction.Blocks[0].Accesses[0].Cost = 0
	case "unknown-value":
		transaction.Blocks[0].Terminator.Condition = 999
	case "helper-cycle":
		helper := &input.StateTransactions[1]
		helper.Accesses = append([]hir.StateAccess{{Kind: hir.StateAccessCall, Callee: helper.Function, Cost: 1}}, helper.Accesses...)
		input.Modules[0].Functions[1].Body.Statements = append([]hir.Statement{{Kind: "expression-statement", Expression: helperCallExpression(helper.Function, input.Modules[0].Functions[1].Effects)}}, input.Modules[0].Functions[1].Body.Statements...)
	case "helper-effect":
		input.Modules[0].Functions[0].Effects = nil
	}
	_, err = publishCompilerResult(input)
	if err == nil || !strings.Contains(err.Error(), mutation.ErrorContains) {
		return &Failure{CaseID: mutation.ID, Detail: fmt.Sprintf("compiler error = %v, want containing %q", err, mutation.ErrorContains)}
	}
	return nil
}

func validateLimits(limits limitsDefinition) error {
	if limits.MaxWrites <= 0 || limits.MaxWriteBytes == 0 || limits.MaxValueBytes <= 0 || limits.MaxMapKeyBytes <= 0 {
		return fmt.Errorf("all limits must be positive")
	}
	return nil
}

func runCase(definition manifest, fixture caseDefinition) *Failure {
	input, err := buildHIRPackage(definition.Layout, fixture)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: err.Error()}
	}
	published, err := publishCompilerResult(input)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("publish compiler result: %v", err)}
	}
	expectedFunctions := 1 + len(fixture.Helpers)
	if !published.Verified || len(published.MIR.Functions) != expectedFunctions || len(published.StateTransactions) != 1 {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf(
			"compiler result = verified %t ordinary functions %d state transactions %d, want true/%d/1",
			published.Verified, len(published.MIR.Functions), len(published.StateTransactions), expectedFunctions,
		)}
	}
	program := published.StateTransactions[0].Program
	store, err := contractstatemir.NewMemoryStore(program.Layout)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("construct store: %v", err)}
	}
	limits := definition.Limits
	if fixture.Limits != nil {
		limits = *fixture.Limits
	}
	outcome, executeErr, recovered := execute(store, program, contractstate.Limits{
		MaxWrites: limits.MaxWrites, MaxWriteBytes: limits.MaxWriteBytes,
		MaxValueBytes: limits.MaxValueBytes, MaxMapKeyBytes: limits.MaxMapKeyBytes,
	})
	if detail := compareResult(fixture, program.Layout, store, outcome, executeErr, recovered); detail != "" {
		return &Failure{CaseID: fixture.ID, Detail: detail}
	}
	return nil
}

func publishCompilerResult(input hir.Package) (compiler.Result, error) {
	return compiler.LowerAnalyzed(compiler.Analysis{Semantic: semantic.Result{
		HIR: input, CompletedThrough: semantic.PhaseFlow,
	}})
}

func buildHIRPackage(layout layoutDefinition, fixture caseDefinition) (hir.Package, error) {
	const packageID hir.PackageID = 1
	const functionID hir.SymbolID = 1
	fields := make([]hir.StateField, 0, len(layout.Fields))
	fieldByName := make(map[string]fieldDefinition, len(layout.Fields))
	for _, field := range layout.Fields {
		fields = append(fields, hir.StateField{
			Name: field.Name, Kind: hir.StateFieldKind(field.Kind),
			KeyTypeID: field.KeyTypeID, ValueTypeID: field.ValueTypeID,
		})
		fieldByName[field.Name] = field
	}
	helperSymbols := make(map[string]hir.SymbolID, len(fixture.Helpers))
	for index, helper := range fixture.Helpers {
		helperSymbols[helper.Name] = hir.SymbolID(index + 2)
	}
	accesses := make([]hir.StateAccess, 0, len(fixture.Operations))
	for index, operation := range fixture.Operations {
		if operation.Kind == "call" {
			accesses = append(accesses, hir.StateAccess{Kind: hir.StateAccessCall, Callee: helperSymbols[operation.Callee], Cost: operation.Cost})
			continue
		}
		field := fieldByName[operation.Field]
		key, err := decodeValue(operation.KeyType, operation.KeyHex)
		if err != nil {
			return hir.Package{}, fmt.Errorf("operation %d key: %w", index, err)
		}
		value, err := decodeValue(operation.ValueType, operation.ValueHex)
		if err != nil {
			return hir.Package{}, fmt.Errorf("operation %d value: %w", index, err)
		}
		if operation.KeyType != "" && operation.KeyType != field.KeyTypeID {
			return hir.Package{}, fmt.Errorf("operation %d key type %q does not match field %q", index, operation.KeyType, field.KeyTypeID)
		}
		if operation.ValueType != "" && operation.ValueType != field.ValueTypeID {
			return hir.Package{}, fmt.Errorf("operation %d value type %q does not match field %q", index, operation.ValueType, field.ValueTypeID)
		}
		accesses = append(accesses, hirStateAccess(operation, key, value))
	}
	blocks := make([]hir.StateBlock, 0, len(fixture.Blocks))
	for _, block := range fixture.Blocks {
		lowered := hir.StateBlock{ID: hir.StateBlockID(block.ID), Parameters: make([]hir.StateBlockParameter, 0, len(block.Parameters)), Accesses: make([]hir.StateAccess, 0, len(block.Operations))}
		for _, parameter := range block.Parameters {
			lowered.Parameters = append(lowered.Parameters, hir.StateBlockParameter{ID: hir.StateValueID(parameter.ID), TypeID: parameter.TypeID})
		}
		for index, operation := range block.Operations {
			if operation.Kind == "call" {
				lowered.Accesses = append(lowered.Accesses, hir.StateAccess{Kind: hir.StateAccessCall, Callee: helperSymbols[operation.Callee], Cost: operation.Cost})
				continue
			}
			field := fieldByName[operation.Field]
			key, err := decodeValue(operation.KeyType, operation.KeyHex)
			if err != nil {
				return hir.Package{}, fmt.Errorf("block %d operation %d key: %w", block.ID, index, err)
			}
			value, err := decodeValue(operation.ValueType, operation.ValueHex)
			if err != nil {
				return hir.Package{}, fmt.Errorf("block %d operation %d value: %w", block.ID, index, err)
			}
			if operation.KeyType != "" && operation.KeyType != field.KeyTypeID {
				return hir.Package{}, fmt.Errorf("block %d operation %d key type mismatch", block.ID, index)
			}
			if operation.ValueType != "" && operation.ValueType != field.ValueTypeID {
				return hir.Package{}, fmt.Errorf("block %d operation %d value type mismatch", block.ID, index)
			}
			lowered.Accesses = append(lowered.Accesses, hirStateAccess(operation, key, value))
		}
		lowered.Terminator = hir.StateTerminator{
			Kind: hir.StateTerminatorKind(block.Terminator.Kind), Condition: hir.StateValueID(block.Terminator.Condition),
			Target: hir.StateBlockID(block.Terminator.Target), Arguments: hirValueIDs(block.Terminator.Arguments),
			TrueTarget: hir.StateBlockID(block.Terminator.TrueTarget), TrueArguments: hirValueIDs(block.Terminator.TrueArguments),
			FalseTarget: hir.StateBlockID(block.Terminator.FalseTarget), FalseArguments: hirValueIDs(block.Terminator.FalseArguments),
			TrapCategory: block.Terminator.TrapCategory, Cost: block.Terminator.Cost,
		}
		blocks = append(blocks, lowered)
	}
	helperEffects := make(map[string][]hir.Effect, len(fixture.Helpers))
	for _, helper := range fixture.Helpers {
		helperEffects[helper.Name] = fixtureBodyEffects(helper.Operations, helper.Blocks, fixture.Helpers, make(map[string]bool))
	}
	functions := []hir.Function{{
		Symbol: functionID, Name: "fixture", Effects: fixtureBodyEffects(fixture.Operations, fixture.Blocks, fixture.Helpers, make(map[string]bool)), ResultType: hir.TypeVoid,
		Body: hir.Block{Statements: fixtureCallStatements(fixture.Operations, fixture.Blocks, helperSymbols, helperEffects)},
	}}
	transactions := []hir.StateTransaction{{
		Package: packageID, Function: functionID, Accesses: accesses,
		Exit: hir.StateTransactionExit(fixture.Exit), TrapCategory: fixture.TrapCategory,
		Entry: hir.StateBlockID(fixture.Entry), Blocks: blocks,
	}}
	for _, helper := range fixture.Helpers {
		helperAccesses := make([]hir.StateAccess, 0, len(helper.Operations))
		for index, operation := range helper.Operations {
			if operation.Kind == "call" {
				helperAccesses = append(helperAccesses, hir.StateAccess{Kind: hir.StateAccessCall, Callee: helperSymbols[operation.Callee], Cost: operation.Cost})
				continue
			}
			field := fieldByName[operation.Field]
			key, err := decodeValue(operation.KeyType, operation.KeyHex)
			if err != nil {
				return hir.Package{}, fmt.Errorf("helper %q operation %d key: %w", helper.Name, index, err)
			}
			value, err := decodeValue(operation.ValueType, operation.ValueHex)
			if err != nil {
				return hir.Package{}, fmt.Errorf("helper %q operation %d value: %w", helper.Name, index, err)
			}
			if operation.KeyType != "" && operation.KeyType != field.KeyTypeID || operation.ValueType != "" && operation.ValueType != field.ValueTypeID {
				return hir.Package{}, fmt.Errorf("helper %q operation %d type mismatch", helper.Name, index)
			}
			helperAccesses = append(helperAccesses, hirStateAccess(operation, key, value))
		}
		symbol := helperSymbols[helper.Name]
		functions = append(functions, hir.Function{
			Symbol: symbol, Name: helper.Name, Effects: helperEffects[helper.Name], ResultType: hir.TypeVoid,
			Body: hir.Block{Statements: fixtureCallStatements(helper.Operations, helper.Blocks, helperSymbols, helperEffects)},
		})
		transactions = append(transactions, hir.StateTransaction{
			Package: packageID, Function: symbol, Accesses: helperAccesses,
			Exit: hir.StateTransactionExit(helper.Exit), TrapCategory: helper.TrapCategory,
		})
	}
	return hir.Package{
		Modules:           []hir.Module{{Package: packageID, Path: "fixture", Functions: functions}},
		StateLayouts:      []hir.StateLayout{{Package: packageID, ContractUID: layout.ContractUID, Fields: fields}},
		StateTransactions: transactions,
	}, nil
}

func fixtureCallStatements(operations []operation, blocks []blockDefinition, symbols map[string]hir.SymbolID, effects map[string][]hir.Effect) []hir.Statement {
	result := make([]hir.Statement, 0)
	appendCalls := func(values []operation) {
		for _, operation := range values {
			if operation.Kind == "call" {
				result = append(result, hir.Statement{Kind: "expression-statement", Expression: helperCallExpression(symbols[operation.Callee], effects[operation.Callee])})
			}
		}
	}
	appendCalls(operations)
	for _, block := range blocks {
		appendCalls(block.Operations)
	}
	return result
}

func helperCallExpression(symbol hir.SymbolID, effects []hir.Effect) *hir.Expression {
	return &hir.Expression{Kind: "call", Type: hir.TypeVoid, Symbol: symbol, Effects: effects, Children: []hir.Expression{{Kind: "path", Symbol: symbol}}}
}

func fixtureBodyEffects(operations []operation, blocks []blockDefinition, helpers []helperDefinition, active map[string]bool) []hir.Effect {
	helperByName := make(map[string]helperDefinition, len(helpers))
	for _, helper := range helpers {
		helperByName[helper.Name] = helper
	}
	var effects []hir.Effect
	appendEffects := func(values []operation) {
		for _, operation := range values {
			switch operation.Kind {
			case "call":
				if !active[operation.Callee] {
					active[operation.Callee] = true
					helper := helperByName[operation.Callee]
					effects = append(effects, fixtureBodyEffects(helper.Operations, helper.Blocks, helpers, active)...)
					delete(active, operation.Callee)
				}
			case "cell-read", "map-read":
				effects = append(effects, hir.EffectStorageRead)
			default:
				effects = append(effects, hir.EffectStorageWrite)
			}
		}
	}
	appendEffects(operations)
	for _, block := range blocks {
		appendEffects(block.Operations)
	}
	return hir.CanonicalEffects(effects)
}

func hirStateAccess(operation operation, key, value hir.StateValue) hir.StateAccess {
	return hir.StateAccess{
		Kind: hir.StateAccessKind(operation.Kind), Field: operation.Field, Key: key, Value: value,
		KeyInput: hir.StateValueID(operation.KeyInput), ValueInput: hir.StateValueID(operation.ValueInput),
		Result: hir.StateValueID(operation.Result), FoundResult: hir.StateValueID(operation.Found), Cost: operation.Cost,
	}
}

func hirValueIDs(values []uint32) []hir.StateValueID {
	result := make([]hir.StateValueID, 0, len(values))
	for _, value := range values {
		result = append(result, hir.StateValueID(value))
	}
	return result
}

func decodeValue(typeID, encodedHex string) (hir.StateValue, error) {
	if typeID == "" && encodedHex == "" {
		return hir.StateValue{}, nil
	}
	if typeID == "" {
		return hir.StateValue{}, fmt.Errorf("encoded bytes have no type id")
	}
	encoded, err := hex.DecodeString(encodedHex)
	if err != nil {
		return hir.StateValue{}, fmt.Errorf("decode hex: %w", err)
	}
	return hir.StateValue{TypeID: typeID, Encoded: encoded}, nil
}

func execute(store *contractstate.MemoryStore, program contractstatemir.Program, limits contractstate.Limits) (outcome contractstatemir.Outcome, err error, recovered any) {
	defer func() { recovered = recover() }()
	outcome, err = contractstatemir.Execute(store, program, limits)
	return outcome, err, nil
}

func compareResult(fixture caseDefinition, layout contractstatemir.Layout, store *contractstate.MemoryStore, outcome contractstatemir.Outcome, executeErr error, recovered any) string {
	expect := fixture.Expect
	switch expect.Status {
	case "trap":
		trap, ok := recovered.(contractstatemir.Trap)
		if !ok || trap.Category != expect.TrapCategory || executeErr != nil {
			return fmt.Sprintf("trap result = recovered %#v error %v, want category %q", recovered, executeErr, expect.TrapCategory)
		}
	case "error":
		if recovered != nil || executeErr == nil || !strings.Contains(executeErr.Error(), expect.ErrorContains) {
			return fmt.Sprintf("error result = recovered %#v error %v, want containing %q", recovered, executeErr, expect.ErrorContains)
		}
	default:
		if recovered != nil || executeErr != nil || string(outcome.Exit) != expect.Exit {
			return fmt.Sprintf("execution result = exit %q recovered %#v error %v, want exit %q", outcome.Exit, recovered, executeErr, expect.Exit)
		}
	}
	if store.Version() != expect.Version || outcome.Receipt.Version != expect.Version || outcome.Receipt.Writes != expect.Writes {
		return fmt.Sprintf("receipt/store = %+v version %d, want version %d writes %d", outcome.Receipt, store.Version(), expect.Version, expect.Writes)
	}
	if outcome.Cost != expect.Cost {
		return fmt.Sprintf("execution cost = %d, want %d", outcome.Cost, expect.Cost)
	}
	if len(outcome.Reads) != len(expect.Reads) {
		return fmt.Sprintf("reads = %d, want %d", len(outcome.Reads), len(expect.Reads))
	}
	for index, expected := range expect.Reads {
		actual := outcome.Reads[index]
		if actual.Operation != expected.Operation || actual.Found != expected.Found || actual.Value.TypeID != expected.TypeID || hex.EncodeToString(actual.Value.Encoded) != expected.ValueHex {
			return fmt.Sprintf("read %d = %+v, want %+v", index, actual, expected)
		}
	}
	entries := store.Dump()
	if len(entries) != len(expect.Store) {
		return fmt.Sprintf("store entries = %d, want %d", len(entries), len(expect.Store))
	}
	fields := make(map[string]contractstatemir.LayoutField, len(layout.Fields))
	for _, field := range layout.Fields {
		fields[field.Name] = field
	}
	for index, expected := range expect.Store {
		field, found := fields[expected.Field]
		expectedKey, err := expectedStorageKey(field, expected.StorageKeyHex)
		if !found || err != nil || entries[index].Key != expectedKey || hex.EncodeToString(entries[index].Value) != expected.ValueHex {
			return fmt.Sprintf("store entry %d = %+v, want field %q value %q", index, entries[index], expected.Field, expected.ValueHex)
		}
	}
	return ""
}

func expectedStorageKey(field contractstatemir.LayoutField, encodedHex string) (contractstate.Key, error) {
	if encodedHex == "" {
		return field.Prefix, nil
	}
	encoded, err := hex.DecodeString(encodedHex)
	if err != nil {
		return contractstate.Key{}, fmt.Errorf("decode expected storage key: %w", err)
	}
	if len(encoded) != len(contractstate.Key{}) {
		return contractstate.Key{}, fmt.Errorf("expected storage key has %d bytes, want %d", len(encoded), len(contractstate.Key{}))
	}
	var key contractstate.Key
	copy(key[:], encoded)
	return key, nil
}
