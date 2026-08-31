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
)

var fixtureSchemaVersions = []string{fixtureSchemaV0, fixtureSchemaV1}

var operationKinds = []string{
	"cell-delete",
	"cell-read",
	"cell-write",
	"map-delete",
	"map-read",
	"map-write",
}

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
	report := Report{CasesRun: len(definition.Cases)}
	for _, fixture := range definition.Cases {
		if failure := runCase(definition, fixture); failure != nil {
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
	for _, fixture := range definition.Cases {
		if fixture.ID == "" {
			return fmt.Errorf("contract-state fixture case has an empty id")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return fmt.Errorf("contract-state fixture case %q is duplicated", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		if !slices.Contains(exitKinds, fixture.Exit) || !slices.Contains(resultStatuses, fixture.Expect.Status) {
			return fmt.Errorf("contract-state fixture case %q has an unsupported exit or status", fixture.ID)
		}
		if fixture.Limits != nil {
			if err := validateLimits(*fixture.Limits); err != nil {
				return fmt.Errorf("contract-state fixture case %q limits: %w", fixture.ID, err)
			}
		}
		for _, operation := range fixture.Operations {
			field, found := fields[operation.Field]
			if !found || !slices.Contains(operationKinds, operation.Kind) || !strings.HasPrefix(operation.Kind, field.Kind+"-") {
				return fmt.Errorf("contract-state fixture case %q has an invalid operation on %q", fixture.ID, operation.Field)
			}
		}
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
	if !published.Verified || len(published.MIR.Functions) != 1 || len(published.StateTransactions) != 1 {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf(
			"compiler result = verified %t ordinary functions %d state transactions %d, want true/1/1",
			published.Verified, len(published.MIR.Functions), len(published.StateTransactions),
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
	accesses := make([]hir.StateAccess, 0, len(fixture.Operations))
	var effects []hir.Effect
	for index, operation := range fixture.Operations {
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
		accesses = append(accesses, hir.StateAccess{
			Kind: hir.StateAccessKind(operation.Kind), Field: operation.Field, Key: key, Value: value,
		})
		if strings.HasSuffix(operation.Kind, "read") {
			effects = append(effects, hir.EffectStorageRead)
		} else {
			effects = append(effects, hir.EffectStorageWrite)
		}
	}
	return hir.Package{
		Modules: []hir.Module{{Package: packageID, Functions: []hir.Function{{
			Symbol: functionID, Name: "fixture", Effects: hir.CanonicalEffects(effects), ResultType: hir.TypeVoid,
		}}}},
		StateLayouts: []hir.StateLayout{{Package: packageID, ContractUID: layout.ContractUID, Fields: fields}},
		StateTransactions: []hir.StateTransaction{{
			Package: packageID, Function: functionID, Accesses: accesses,
			Exit: hir.StateTransactionExit(fixture.Exit), TrapCategory: fixture.TrapCategory,
		}},
	}, nil
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
