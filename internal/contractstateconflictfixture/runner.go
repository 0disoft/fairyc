package contractstateconflictfixture

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
)

const fixtureSchemaVersion = "fairyc.contract-state-conflict-fixtures/v0"

var actionKinds = []string{"begin", "cell-read", "cell-write", "commit", "rollback"}
var resultStatuses = []string{"commit", "conflict", "ok", "open", "rolled-back"}

type fixtureLayout struct {
	layout contractstate.Layout
	cells  map[string]contractstate.EncodedCell
}

// Run executes every schedule against a fresh store with explicitly overlapping transactions.
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
	if definition.Schema == "" || definition.SchemaVersion != fixtureSchemaVersion || len(definition.Contracts) == 0 {
		return fmt.Errorf("invalid contract-state conflict fixture contract header")
	}
	if definition.Layout.ContractUID == "" || len(definition.Layout.Fields) == 0 {
		return fmt.Errorf("contract-state conflict fixture layout is empty")
	}
	if definition.Limits.MaxWrites <= 0 || definition.Limits.MaxWriteBytes == 0 || definition.Limits.MaxValueBytes <= 0 || definition.Limits.MaxMapKeyBytes <= 0 {
		return fmt.Errorf("contract-state conflict fixture limits must be positive")
	}
	fields := make(map[string]fieldDefinition, len(definition.Layout.Fields))
	for _, field := range definition.Layout.Fields {
		if field.Name == "" || field.ValueTypeID == "" {
			return fmt.Errorf("contract-state conflict fixture field has an empty identity")
		}
		if _, duplicate := fields[field.Name]; duplicate {
			return fmt.Errorf("contract-state conflict fixture field %q is duplicated", field.Name)
		}
		fields[field.Name] = field
	}
	seenCases := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if fixture.ID == "" || len(fixture.Steps) == 0 {
			return fmt.Errorf("contract-state conflict fixture case has an empty identity or schedule")
		}
		if _, duplicate := seenCases[fixture.ID]; duplicate {
			return fmt.Errorf("contract-state conflict fixture case %q is duplicated", fixture.ID)
		}
		seenCases[fixture.ID] = struct{}{}
		seenSteps := make(map[string]struct{}, len(fixture.Steps))
		for _, step := range fixture.Steps {
			if step.ID == "" || step.Transaction == "" || !slices.Contains(actionKinds, step.Action) || !slices.Contains(resultStatuses, step.Expect.Status) {
				return fmt.Errorf("contract-state conflict fixture case %q has an invalid step", fixture.ID)
			}
			if _, duplicate := seenSteps[step.ID]; duplicate {
				return fmt.Errorf("contract-state conflict fixture case %q step %q is duplicated", fixture.ID, step.ID)
			}
			seenSteps[step.ID] = struct{}{}
			if step.Action == "cell-read" || step.Action == "cell-write" {
				field, found := fields[step.Field]
				if !found || (step.ValueTypeID != "" && step.ValueTypeID != field.ValueTypeID) {
					return fmt.Errorf("contract-state conflict fixture case %q step %q has an invalid field or type", fixture.ID, step.ID)
				}
			}
			if err := validateStepShape(step); err != nil {
				return fmt.Errorf("contract-state conflict fixture case %q step %q: %w", fixture.ID, step.ID, err)
			}
		}
		seenStoreFields := make(map[string]struct{}, len(fixture.Expect.Store))
		for _, entry := range fixture.Expect.Store {
			if _, found := fields[entry.Field]; !found {
				return fmt.Errorf("contract-state conflict fixture case %q expects unknown field %q", fixture.ID, entry.Field)
			}
			if _, duplicate := seenStoreFields[entry.Field]; duplicate {
				return fmt.Errorf("contract-state conflict fixture case %q repeats store field %q", fixture.ID, entry.Field)
			}
			seenStoreFields[entry.Field] = struct{}{}
		}
	}
	return nil
}

func validateStepShape(step stepDefinition) error {
	switch step.Action {
	case "begin":
		if step.Expect.Status != "open" || step.Field != "" || step.ValueTypeID != "" || step.ValueHex != "" || step.Expect.Found != nil {
			return fmt.Errorf("begin must declare only an open result")
		}
	case "cell-read":
		if step.Expect.Status != "ok" || step.Field == "" || step.ValueTypeID != "" || step.ValueHex != "" || step.Expect.Found == nil {
			return fmt.Errorf("cell-read must declare a field and exact read result")
		}
	case "cell-write":
		if step.Expect.Status != "ok" || step.Field == "" || step.ValueTypeID == "" || step.ValueHex == "" || step.Expect.Found != nil {
			return fmt.Errorf("cell-write must declare one typed value and an ok result")
		}
	case "commit":
		if (step.Expect.Status != "commit" && step.Expect.Status != "conflict") || step.Field != "" || step.ValueTypeID != "" || step.ValueHex != "" || step.Expect.Found != nil {
			return fmt.Errorf("commit must declare only a commit or conflict result")
		}
		if step.Expect.Status == "conflict" && (step.Expect.Version != 0 || step.Expect.Writes != 0) {
			return fmt.Errorf("conflict cannot declare a commit receipt")
		}
	case "rollback":
		if step.Expect.Status != "rolled-back" || step.Field != "" || step.ValueTypeID != "" || step.ValueHex != "" || step.Expect.Found != nil {
			return fmt.Errorf("rollback must declare only a rolled-back result")
		}
	default:
		return fmt.Errorf("unsupported action %q", step.Action)
	}
	return nil
}

func runCase(definition manifest, fixture caseDefinition) *Failure {
	bound, err := buildLayout(definition.Layout)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: err.Error()}
	}
	store, err := contractstate.NewMemoryStore(bound.layout)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("construct store: %v", err)}
	}
	limits := contractstate.Limits{
		MaxWrites: definition.Limits.MaxWrites, MaxWriteBytes: definition.Limits.MaxWriteBytes,
		MaxValueBytes: definition.Limits.MaxValueBytes, MaxMapKeyBytes: definition.Limits.MaxMapKeyBytes,
	}
	transactions := make(map[string]*contractstate.Transaction)
	usedTransactions := make(map[string]struct{})
	for _, step := range fixture.Steps {
		if detail := executeStep(step, bound, store, limits, transactions, usedTransactions); detail != "" {
			return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("step %q: %s", step.ID, detail)}
		}
	}
	if len(transactions) != 0 {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("schedule leaves %d transaction(s) open", len(transactions))}
	}
	if detail := compareFinal(fixture.Expect, bound, store); detail != "" {
		return &Failure{CaseID: fixture.ID, Detail: detail}
	}
	return nil
}

func buildLayout(definition layoutDefinition) (fixtureLayout, error) {
	builder, err := contractstate.NewLayoutBuilder(definition.ContractUID)
	if err != nil {
		return fixtureLayout{}, err
	}
	cells := make(map[string]contractstate.EncodedCell, len(definition.Fields))
	for _, field := range definition.Fields {
		cell, err := contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
		if err != nil {
			return fixtureLayout{}, err
		}
		cells[field.Name] = cell
	}
	layout, err := builder.Build()
	if err != nil {
		return fixtureLayout{}, err
	}
	return fixtureLayout{layout: layout, cells: cells}, nil
}

func executeStep(step stepDefinition, bound fixtureLayout, store *contractstate.MemoryStore, limits contractstate.Limits, transactions map[string]*contractstate.Transaction, used map[string]struct{}) string {
	switch step.Action {
	case "begin":
		if _, duplicate := used[step.Transaction]; duplicate {
			return "transaction identity is reused"
		}
		transaction, err := store.Begin(bound.layout, limits)
		if err != nil {
			return err.Error()
		}
		transactions[step.Transaction] = transaction
		used[step.Transaction] = struct{}{}
		return expectStatus(step.Expect, "open", contractstate.CommitReceipt{}, nil)
	case "cell-read":
		transaction, detail := openTransaction(step.Transaction, transactions)
		if detail != "" {
			return detail
		}
		value, found, err := bound.cells[step.Field].Get(transaction)
		if err != nil {
			return err.Error()
		}
		if detail := expectStatus(step.Expect, "ok", contractstate.CommitReceipt{}, nil); detail != "" {
			return detail
		}
		if step.Expect.Found == nil || found != *step.Expect.Found || hex.EncodeToString(value) != step.Expect.ValueHex {
			return fmt.Sprintf("read = found %t value %x, want found %v value %s", found, value, step.Expect.Found, step.Expect.ValueHex)
		}
		return ""
	case "cell-write":
		transaction, detail := openTransaction(step.Transaction, transactions)
		if detail != "" {
			return detail
		}
		value, err := hex.DecodeString(step.ValueHex)
		if err != nil {
			return fmt.Sprintf("decode value: %v", err)
		}
		if err := bound.cells[step.Field].Set(transaction, value); err != nil {
			return err.Error()
		}
		return expectStatus(step.Expect, "ok", contractstate.CommitReceipt{}, nil)
	case "commit":
		transaction, detail := openTransaction(step.Transaction, transactions)
		if detail != "" {
			return detail
		}
		receipt, err := transaction.Commit()
		delete(transactions, step.Transaction)
		actual := "commit"
		if errors.Is(err, contractstate.ErrConflict) {
			actual = "conflict"
		} else if err != nil {
			return err.Error()
		}
		return expectStatus(step.Expect, actual, receipt, err)
	case "rollback":
		transaction, detail := openTransaction(step.Transaction, transactions)
		if detail != "" {
			return detail
		}
		closed := transaction.Rollback()
		delete(transactions, step.Transaction)
		if !closed {
			return "rollback did not close the transaction"
		}
		return expectStatus(step.Expect, "rolled-back", contractstate.CommitReceipt{}, nil)
	default:
		return fmt.Sprintf("unsupported action %q", step.Action)
	}
}

func openTransaction(name string, transactions map[string]*contractstate.Transaction) (*contractstate.Transaction, string) {
	transaction, found := transactions[name]
	if !found {
		return nil, "transaction is not open"
	}
	return transaction, ""
}

func expectStatus(expect stepExpectation, actual string, receipt contractstate.CommitReceipt, err error) string {
	if expect.Status != actual {
		return fmt.Sprintf("status = %q, want %q", actual, expect.Status)
	}
	if actual == "commit" && (receipt.Version != expect.Version || receipt.Writes != expect.Writes) {
		return fmt.Sprintf("receipt = %+v, want version %d writes %d", receipt, expect.Version, expect.Writes)
	}
	if actual == "conflict" && (!errors.Is(err, contractstate.ErrConflict) || receipt != (contractstate.CommitReceipt{})) {
		return fmt.Sprintf("conflict = receipt %+v error %v", receipt, err)
	}
	return ""
}

func compareFinal(expect finalExpectation, bound fixtureLayout, store *contractstate.MemoryStore) string {
	if store.Version() != expect.Version {
		return fmt.Sprintf("store version = %d, want %d", store.Version(), expect.Version)
	}
	actual := make(map[contractstate.Key][]byte)
	for _, entry := range store.Dump() {
		actual[entry.Key] = entry.Value
	}
	if len(actual) != len(expect.Store) {
		return fmt.Sprintf("store entries = %d, want %d", len(actual), len(expect.Store))
	}
	for _, expected := range expect.Store {
		cell, found := bound.cells[expected.Field]
		value, present := actual[cell.Field().Prefix]
		if !found || !present || hex.EncodeToString(value) != expected.ValueHex {
			return fmt.Sprintf("store field %q = %x present %t, want %s", expected.Field, value, present, expected.ValueHex)
		}
	}
	return ""
}
