package mirverifyfixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/0disoft/fairyc/internal/mir"
)

const verifierFixtureSchema = "fairyc.mir-verifier-fixtures/v0"

// Run constructs every named seed, verifies its baseline, applies one mutation, and verifies the rejection.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	report := Report{SeedsRun: len(definition.Seeds), CasesRun: len(definition.Cases)}
	seedDefinitions := make(map[string]seedDefinition, len(definition.Seeds))
	baselines := make(map[string]mir.Program, len(definition.Seeds))
	for _, definition := range definition.Seeds {
		if _, duplicate := seedDefinitions[definition.ID]; duplicate {
			report.Failures = append(report.Failures, Failure{CaseID: definition.ID, Detail: "duplicate seed id"})
			continue
		}
		seedDefinitions[definition.ID] = definition
		baseline, failure := validateSeed(definition)
		if failure != nil {
			report.Failures = append(report.Failures, *failure)
			continue
		}
		baselines[definition.ID] = baseline
	}

	seenCases := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if _, duplicate := seenCases[fixture.ID]; duplicate {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "duplicate case id"})
			continue
		}
		seenCases[fixture.ID] = struct{}{}
		baseline, found := baselines[fixture.Seed]
		if !found {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("seed %q is unavailable", fixture.Seed)})
			continue
		}
		report.Failures = append(report.Failures, runCase(fixture, baseline)...)
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
	if definition.Schema == "" || definition.SchemaVersion != verifierFixtureSchema || len(definition.Contracts) == 0 {
		return manifest{}, fmt.Errorf("invalid MIR verifier fixture contract header")
	}
	return definition, nil
}

func validateSeed(definition seedDefinition) (mir.Program, *Failure) {
	if definition.Baseline != "verify" || !sortedUnique(definition.Entities) {
		return mir.Program{}, &Failure{CaseID: definition.ID, Detail: "seed does not declare a sorted verified entity set"}
	}
	program, err := constructSeed(definition.ID)
	if err != nil {
		return mir.Program{}, &Failure{CaseID: definition.ID, Detail: err.Error()}
	}
	actualEntities := seedEntities(program)
	slices.Sort(actualEntities)
	if !slices.Equal(actualEntities, definition.Entities) {
		return mir.Program{}, &Failure{CaseID: definition.ID, Detail: fmt.Sprintf("entity set: want %v, got %v", definition.Entities, actualEntities)}
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return mir.Program{}, &Failure{CaseID: definition.ID, Detail: fmt.Sprintf("baseline verifier failure: %+v", failures)}
	}
	return program, nil
}

func runCase(fixture caseDefinition, baseline mir.Program) []Failure {
	if fixture.Expect.Status != "reject" || len(fixture.Expect.Codes) != 1 || fixture.Expect.PrimaryEntity == "" {
		return []Failure{{CaseID: fixture.ID, Detail: "case does not declare one verifier rejection"}}
	}
	if err := validateMutationShape(fixture.Mutation); err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	entities := seedEntities(baseline)
	if !slices.Contains(entities, fixture.Mutation.Target) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("mutation target %q is absent", fixture.Mutation.Target)}}
	}
	if fixture.Mutation.SupportingEntity != "" && !slices.Contains(entities, fixture.Mutation.SupportingEntity) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("supporting entity %q is absent", fixture.Mutation.SupportingEntity)}}
	}
	mutated := mir.Clone(baseline)
	applied, err := applyMutation(&mutated, fixture.Mutation)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	if applied != 1 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("mutation applied %d times, want exactly 1", applied)}}
	}
	actual := mir.Verify(mutated)
	if len(actual) != 1 || actual[0].Code != fixture.Expect.Codes[0] || actual[0].Entity != fixture.Expect.PrimaryEntity {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("verifier result: want %s at %s, got %+v", fixture.Expect.Codes[0], fixture.Expect.PrimaryEntity, actual)}}
	}
	return nil
}

func validateMutationShape(mutation mutation) error {
	expectedSupporting := ""
	switch mutation.Kind {
	case "allocate-needs-drop-in-plain-region":
		expectedSupporting = "local:resource-value"
	case "remove-required-trap-check":
		expectedSupporting = "operation:zero-check"
	}
	if mutation.SupportingEntity != expectedSupporting {
		return fmt.Errorf("mutation %q supporting entity: want %q, got %q", mutation.Kind, expectedSupporting, mutation.SupportingEntity)
	}
	return nil
}

func applyMutation(program *mir.Program, mutation mutation) (int, error) {
	if len(program.Functions) != 1 {
		return 0, fmt.Errorf("mutation seed must contain exactly one function")
	}
	function := &program.Functions[0]
	switch mutation.Kind {
	case "remove-terminator":
		if terminator := findTerminator(function, mutation.Target); terminator != nil {
			*terminator = mir.Terminator{UnwindSuccessor: mir.NoBlock}
			return 1, nil
		}
	case "use-before-initialize":
		useBlock, useIndex := findOperation(function, mutation.Target)
		initBlock, initIndex := findOperation(function, "statement:init-value")
		if useBlock != nil && useBlock == initBlock && initIndex < useIndex {
			useBlock.Operations[initIndex], useBlock.Operations[useIndex] = useBlock.Operations[useIndex], useBlock.Operations[initIndex]
			return 1, nil
		}
	case "duplicate-resource-discharge":
		block, index := findOperation(function, "statement:drop-token")
		if block != nil {
			duplicate := mir.Operation{Kind: mir.OperationMove, Entity: "mutation:duplicate-discharge", Symbol: tokenSymbol}
			block.Operations = insertOperation(block.Operations, index+1, duplicate)
			return 1, nil
		}
	case "remove-exit-cleanup":
		if block, index := findOperation(function, mutation.Target); block != nil {
			block.Operations = append(block.Operations[:index], block.Operations[index+1:]...)
			return 1, nil
		}
	case "move-resource-projection":
		if block, index := findOperation(function, "statement:drop-token"); block != nil {
			block.Operations[index] = mir.Operation{
				Kind: mir.OperationMove, Entity: "statement:drop-token", Symbol: tokenSymbol, ProjectionEntity: mutation.Target,
			}
			return 1, nil
		}
	case "remove-flag-initialization":
		if block, index := findOperation(function, mutation.Target); block != nil {
			block.Operations = append(block.Operations[:index], block.Operations[index+1:]...)
			return 1, nil
		}
	case "drop-before-flag-clear":
		clearBlock, clearIndex := findOperation(function, mutation.Target)
		dropBlock, dropIndex := findOperation(function, "statement:drop-token")
		if clearBlock != nil && clearBlock == dropBlock && clearIndex < dropIndex {
			clearBlock.Operations[clearIndex], clearBlock.Operations[dropIndex] = clearBlock.Operations[dropIndex], clearBlock.Operations[clearIndex]
			return 1, nil
		}
	case "extend-loan-past-origin":
		for index := range function.Loans {
			if function.Loans[index].Entity == mutation.Target {
				function.Loans[index].OutlivesOrigin = true
				return 1, nil
			}
		}
	case "add-return-origin":
		for index := range function.ReturnOrigins {
			if function.ReturnOrigins[index].Entity == mutation.Target {
				function.ReturnOrigins[index].Returned = true
				return 1, nil
			}
		}
	case "escape-region-value":
		for index := range function.RegionValues {
			if function.RegionValues[index].Entity == mutation.Target {
				function.RegionValues[index].Escapes = true
				return 1, nil
			}
		}
	case "allocate-needs-drop-in-plain-region":
		for index := range function.RegionValues {
			if function.RegionValues[index].OperationEntity == mutation.Target {
				function.RegionValues[index].NeedsDrop = true
				return 1, nil
			}
		}
	case "remove-required-trap-check":
		if block, index := findOperation(function, mutation.Target); block != nil {
			block.Operations[index].TrapCategories = []string{"signed-division-overflow"}
			return 1, nil
		}
	case "add-unwind-successor":
		if terminator := findTerminator(function, mutation.Target); terminator != nil && len(terminator.Successors) == 1 {
			terminator.UnwindSuccessor = terminator.Successors[0]
			return 1, nil
		}
	case "insert-source-form":
		if block, index := findOperation(function, mutation.Target); block != nil {
			block.Operations[index].SourceForm = "question"
			return 1, nil
		}
	case "remove-unsafe-provenance":
		for index := range function.UnsafeOperations {
			if function.UnsafeOperations[index].Entity == mutation.Target {
				function.UnsafeOperations[index].SourceSpanEntity = ""
				function.UnsafeOperations[index].ScopeEntity = ""
				return 1, nil
			}
		}
	default:
		return 0, fmt.Errorf("unsupported mutation kind %q", mutation.Kind)
	}
	return 0, fmt.Errorf("mutation %q could not be applied to %q", mutation.Kind, mutation.Target)
}

func findOperation(function *mir.Function, entity string) (*mir.Block, int) {
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			if function.Blocks[blockIndex].Operations[operationIndex].Entity == entity {
				return &function.Blocks[blockIndex], operationIndex
			}
		}
	}
	return nil, -1
}

func findTerminator(function *mir.Function, entity string) *mir.Terminator {
	for blockIndex := range function.Blocks {
		if function.Blocks[blockIndex].Terminator.Entity == entity {
			return &function.Blocks[blockIndex].Terminator
		}
	}
	return nil
}

func insertOperation(values []mir.Operation, index int, value mir.Operation) []mir.Operation {
	values = append(values, mir.Operation{})
	copy(values[index+1:], values[index:])
	values[index] = value
	return values
}

func sortedUnique(values []string) bool {
	if len(values) == 0 || !slices.IsSorted(values) {
		return false
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] == values[index] {
			return false
		}
	}
	return true
}
