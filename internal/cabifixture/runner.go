// Package cabifixture executes the versioned Stage 0 C ABI classification fixture contract.
package cabifixture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/cabi"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	fixtureSchema    = "fairyc.c-abi-fixtures/v0"
	maxManifestBytes = 1024 * 1024
)

// Failure is one case-local contract mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report summarizes one complete fixture manifest execution.
type Report struct {
	Target   target.ID
	CasesRun int
	Passed   int
	Failures []Failure
}

// NativeProbeCase is one validated pass case safe to translate into a bidirectional C probe.
type NativeProbeCase struct {
	ID        string
	Signature cabi.Signature
	Plan      cabi.Plan
}

type manifest struct {
	Schema        string        `json:"$schema"`
	SchemaVersion string        `json:"schemaVersion"`
	Target        string        `json:"target"`
	Contracts     []string      `json:"contracts"`
	Deferred      []string      `json:"deferred"`
	Cases         []fixtureCase `json:"cases"`
}

type fixtureCase struct {
	ID        string      `json:"id"`
	Signature signature   `json:"signature"`
	Expect    expectation `json:"expect"`
}

type signature struct {
	Result     *layout.Type  `json:"result"`
	Parameters []layout.Type `json:"parameters"`
}

type expectation struct {
	Status    string         `json:"status"`
	Plan      *observedPlan  `json:"plan,omitempty"`
	ErrorCode cabi.ErrorCode `json:"errorCode,omitempty"`
}

type observedPlan struct {
	Return    observedLocation   `json:"return"`
	Arguments []observedLocation `json:"arguments"`
}

type observedLocation struct {
	Kind                  cabi.LocationKind `json:"kind"`
	Classes               []cabi.Class      `json:"classes"`
	Registers             []cabi.Register   `json:"registers"`
	HiddenPointerRegister cabi.Register     `json:"hiddenPointerRegister,omitempty"`
	PointerResultRegister cabi.Register     `json:"pointerResultRegister,omitempty"`
}

// Run decodes, semantically validates, and executes every ABI case in manifestPath.
func Run(manifestPath string) (Report, error) {
	definition, err := loadManifest(manifestPath)
	if err != nil {
		return Report{}, err
	}
	report, _, err := executeManifest(definition)
	return report, err
}

// NativeProbeCases returns isolated copies of all validated pass cases in canonical order.
func NativeProbeCases(manifestPath string) (target.ID, []NativeProbeCase, error) {
	definition, err := loadManifest(manifestPath)
	if err != nil {
		return "", nil, err
	}
	report, cases, err := executeManifest(definition)
	if err != nil {
		return "", nil, err
	}
	if len(report.Failures) != 0 || report.Passed != report.CasesRun {
		first := "case count mismatch"
		if len(report.Failures) != 0 {
			first = report.Failures[0].CaseID + ": " + report.Failures[0].Detail
		}
		return "", nil, fmt.Errorf("C ABI fixture manifest is not clean: %s", first)
	}
	return report.Target, cases, nil
}

func executeManifest(definition manifest) (Report, []NativeProbeCase, error) {
	targetID := target.ID(definition.Target)
	spec, found := target.Lookup(targetID)
	if !found {
		return Report{}, nil, fmt.Errorf("C ABI fixture target %q is unknown", definition.Target)
	}
	engine, err := cabi.New(spec)
	if err != nil {
		return Report{}, nil, err
	}
	report := Report{Target: targetID, CasesRun: len(definition.Cases), Failures: []Failure{}}
	probeCases := make([]NativeProbeCase, 0, len(definition.Cases))
	seen := make(map[string]struct{}, len(definition.Cases))
	previousID := ""
	for _, fixture := range definition.Cases {
		if fixture.ID == "" {
			report.Failures = append(report.Failures, Failure{Detail: "case id is empty"})
			continue
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "duplicate case id"})
			continue
		}
		seen[fixture.ID] = struct{}{}
		if previousID != "" && previousID >= fixture.ID {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "case ids are not strictly sorted"})
			continue
		}
		previousID = fixture.ID
		plan, failure := executeCase(engine, fixture)
		if failure != nil {
			report.Failures = append(report.Failures, *failure)
			continue
		}
		report.Passed++
		if fixture.Expect.Status == "pass" {
			probeCases = append(probeCases, NativeProbeCase{
				ID: fixture.ID, Signature: cloneSignature(fixture.Signature), Plan: clonePlan(plan),
			})
		}
	}
	return report, probeCases, nil
}

func executeCase(engine *cabi.Engine, fixture fixtureCase) (cabi.Plan, *Failure) {
	if !strings.HasPrefix(fixture.ID, fixture.Expect.Status+".") {
		return cabi.Plan{}, &Failure{CaseID: fixture.ID, Detail: "case id prefix does not match expectation status"}
	}
	plan, err := engine.Plan(cabi.Signature{Result: fixture.Signature.Result, Parameters: fixture.Signature.Parameters})
	switch fixture.Expect.Status {
	case "pass":
		if fixture.Expect.Plan == nil || fixture.Expect.ErrorCode != "" {
			return cabi.Plan{}, &Failure{CaseID: fixture.ID, Detail: "pass expectation must contain only plan"}
		}
		if err != nil {
			return cabi.Plan{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("classification failed: %v", err)}
		}
		actual := observePlan(plan)
		if !reflect.DeepEqual(actual, *fixture.Expect.Plan) {
			return cabi.Plan{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("plan mismatch: got %+v, want %+v", actual, *fixture.Expect.Plan)}
		}
	case "fail":
		if fixture.Expect.Plan != nil || fixture.Expect.ErrorCode == "" {
			return cabi.Plan{}, &Failure{CaseID: fixture.ID, Detail: "fail expectation must contain only errorCode"}
		}
		if err == nil {
			return cabi.Plan{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("classification succeeded unexpectedly: %+v", observePlan(plan))}
		}
		code, ok := cabi.Code(err)
		if !ok || code != fixture.Expect.ErrorCode {
			return cabi.Plan{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("error code mismatch: got %q from %v, want %q", code, err, fixture.Expect.ErrorCode)}
		}
	default:
		return cabi.Plan{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unknown expectation status %q", fixture.Expect.Status)}
	}
	return plan, nil
}

func observePlan(plan cabi.Plan) observedPlan {
	result := observedPlan{
		Return: observedLocation{
			Kind: plan.Return.Kind, Classes: slices.Clone(plan.Return.Classification.Classes),
			Registers: slices.Clone(plan.Return.Registers), HiddenPointerRegister: plan.Return.HiddenPointerRegister,
			PointerResultRegister: plan.Return.PointerResultRegister,
		},
		Arguments: make([]observedLocation, len(plan.Arguments)),
	}
	for index, argument := range plan.Arguments {
		result.Arguments[index] = observedLocation{
			Kind: argument.Kind, Classes: slices.Clone(argument.Classification.Classes), Registers: slices.Clone(argument.Registers),
		}
	}
	return result
}

func cloneSignature(value signature) cabi.Signature {
	result := cabi.Signature{Parameters: make([]layout.Type, len(value.Parameters))}
	if value.Result != nil {
		cloned := cloneType(*value.Result)
		result.Result = &cloned
	}
	for index, parameter := range value.Parameters {
		result.Parameters[index] = cloneType(parameter)
	}
	return result
}

func cloneType(value layout.Type) layout.Type {
	result := value
	if value.Element != nil {
		element := cloneType(*value.Element)
		result.Element = &element
	}
	result.Fields = make([]layout.Field, len(value.Fields))
	for index, field := range value.Fields {
		result.Fields[index] = layout.Field{Name: field.Name, Type: cloneType(field.Type)}
	}
	return result
}

func clonePlan(value cabi.Plan) cabi.Plan {
	result := value
	result.Return.Classification.Classes = slices.Clone(value.Return.Classification.Classes)
	result.Return.Registers = slices.Clone(value.Return.Registers)
	result.Arguments = make([]cabi.ArgumentLocation, len(value.Arguments))
	for index, argument := range value.Arguments {
		result.Arguments[index] = argument
		result.Arguments[index].Classification.Classes = slices.Clone(argument.Classification.Classes)
		result.Arguments[index].Registers = slices.Clone(argument.Registers)
	}
	return result
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open C ABI manifest: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return manifest{}, fmt.Errorf("inspect C ABI manifest: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxManifestBytes {
		return manifest{}, fmt.Errorf("C ABI manifest has invalid size %d", info.Size())
	}
	contents, err := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	if err != nil {
		return manifest{}, fmt.Errorf("read C ABI manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var value manifest
	if err := decoder.Decode(&value); err != nil {
		return manifest{}, fmt.Errorf("decode C ABI manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return manifest{}, fmt.Errorf("decode C ABI manifest trailing data")
	}
	if value.Schema != "./schema-v0.json" {
		return manifest{}, fmt.Errorf("C ABI manifest schema reference is %q, want %q", value.Schema, "./schema-v0.json")
	}
	if value.SchemaVersion != fixtureSchema {
		return manifest{}, fmt.Errorf("C ABI manifest schema is %q, want %q", value.SchemaVersion, fixtureSchema)
	}
	if strings.TrimSpace(value.Target) == "" || value.Target != strings.TrimSpace(value.Target) || len(value.Contracts) == 0 || len(value.Deferred) == 0 || len(value.Cases) == 0 {
		return manifest{}, fmt.Errorf("C ABI manifest is missing target, contracts, deferred coverage, or cases")
	}
	if err := validateStringList("contract", value.Contracts); err != nil {
		return manifest{}, err
	}
	if err := validateStringList("deferred coverage", value.Deferred); err != nil {
		return manifest{}, err
	}
	return value, nil
}

func validateStringList(name string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			return fmt.Errorf("C ABI manifest %s contains an empty or padded value", name)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("C ABI manifest %s %q is repeated", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}
