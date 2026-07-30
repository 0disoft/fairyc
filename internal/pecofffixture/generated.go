package pecofffixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/0disoft/fairyc/internal/pecoff"
)

const generatedFixtureSchema = "fairyc.coff-generated-object-fixtures/v0"

type generatedManifest struct {
	Schema        string                 `json:"$schema"`
	SchemaVersion string                 `json:"schemaVersion"`
	Target        string                 `json:"target"`
	Contracts     []string               `json:"contracts"`
	Deferred      []string               `json:"deferred"`
	Cases         []generatedFixtureCase `json:"cases"`
}

type generatedFixtureCase struct {
	ID        string               `json:"id"`
	Generator string               `json:"generator"`
	Inputs    []string             `json:"inputs"`
	Expect    generatedExpectation `json:"expect"`
}

type generatedExpectation struct {
	ByteLength       int                              `json:"byteLength"`
	SHA256           string                           `json:"sha256"`
	Machine          uint16                           `json:"machine"`
	Characteristics  uint16                           `json:"characteristics"`
	BigObject        bool                             `json:"bigObject"`
	BigObjectVersion uint16                           `json:"bigObjectVersion"`
	Sections         []sectionExpectation             `json:"sections"`
	Symbols          []generatedSymbolExpectation     `json:"symbols"`
	RelocationRuns   []generatedRelocationExpectation `json:"relocationRuns"`
}

type generatedSymbolExpectation struct {
	TableIndex        uint32 `json:"tableIndex"`
	Name              string `json:"name"`
	Value             uint32 `json:"value"`
	SectionNumber     int32  `json:"sectionNumber"`
	Type              uint16 `json:"type"`
	StorageClass      uint8  `json:"storageClass"`
	AuxiliaryRecords  uint8  `json:"auxiliaryRecords"`
	WeakFallbackIndex uint32 `json:"weakFallbackIndex"`
	WeakFallback      string `json:"weakFallback"`
	WeakSearch        uint32 `json:"weakSearch"`
}

type generatedRelocationExpectation struct {
	Start          int    `json:"start"`
	Count          int    `json:"count"`
	SectionIndex   int    `json:"sectionIndex"`
	Section        string `json:"section"`
	VirtualAddress uint32 `json:"virtualAddress"`
	SymbolIndex    uint32 `json:"symbolIndex"`
	Symbol         string `json:"symbol"`
	Type           uint16 `json:"type"`
}

// RunGenerated executes compact, versioned contracts for deterministic COFF object generators.
// The manifest freezes exact bytes by length and SHA-256 while relocation runs cover every parsed
// relocation without storing a multi-megabyte hexadecimal fixture.
func RunGenerated(manifestPath string) (Report, error) {
	definition, err := loadGeneratedManifest(manifestPath)
	if err != nil {
		return Report{}, err
	}
	report := Report{Target: definition.Target, CasesRun: len(definition.Cases), Failures: []Failure{}}
	previousID := ""
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if !validCaseID(fixture.ID) {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "case id must match pass.[a-z0-9-]+"})
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
		if failure := executeGeneratedCase(fixture); failure != nil {
			report.Failures = append(report.Failures, *failure)
			continue
		}
		report.Passed++
	}
	return report, nil
}

func executeGeneratedCase(fixture generatedFixtureCase) *Failure {
	object, err := buildGeneratedObject(fixture.Generator, fixture.Inputs)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("generate object: %v", err)}
	}
	replay, err := buildGeneratedObject(fixture.Generator, fixture.Inputs)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("replay object generation: %v", err)}
	}
	if !bytes.Equal(object, replay) {
		return &Failure{CaseID: fixture.ID, Detail: "generator is not byte deterministic"}
	}
	if len(object) != fixture.Expect.ByteLength {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("byte length mismatch: got %d, want %d", len(object), fixture.Expect.ByteLength)}
	}
	digest := sha256.Sum256(object)
	actualSHA := hex.EncodeToString(digest[:])
	if actualSHA != fixture.Expect.SHA256 {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("SHA-256 mismatch: got %s, want %s", actualSHA, fixture.Expect.SHA256)}
	}
	objectReport, err := pecoff.InspectObject(object)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("inspect generated object: %v", err)}
	}
	if objectReport.Machine != fixture.Expect.Machine || objectReport.Characteristics != fixture.Expect.Characteristics ||
		objectReport.BigObject != fixture.Expect.BigObject || objectReport.BigObjectVersion != fixture.Expect.BigObjectVersion {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("object identity mismatch: got machine=%d characteristics=%d bigObject=%t version=%d",
			objectReport.Machine, objectReport.Characteristics, objectReport.BigObject, objectReport.BigObjectVersion)}
	}
	sections := make([]sectionExpectation, len(objectReport.Sections))
	for index, section := range objectReport.Sections {
		sections[index] = sectionExpectation{Name: section.Name, Relocations: section.Relocations, Characteristics: section.Characteristics,
			COMDATSelection: section.COMDATSelection, AssociativeSection: section.AssociativeSection}
	}
	if !reflect.DeepEqual(sections, fixture.Expect.Sections) {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("section receipt mismatch: got %+v, want %+v", sections, fixture.Expect.Sections)}
	}
	symbols := make([]generatedSymbolExpectation, len(objectReport.SymbolRecords))
	for index, symbol := range objectReport.SymbolRecords {
		symbols[index] = generatedSymbolExpectation{
			TableIndex: symbol.TableIndex, Name: symbol.Name, Value: symbol.Value, SectionNumber: symbol.SectionNumber,
			Type: symbol.Type, StorageClass: symbol.StorageClass, AuxiliaryRecords: symbol.AuxiliaryRecords,
			WeakFallbackIndex: symbol.WeakFallbackIndex, WeakFallback: symbol.WeakFallback, WeakSearch: symbol.WeakSearch,
		}
	}
	if !reflect.DeepEqual(symbols, fixture.Expect.Symbols) {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("symbol receipt mismatch: got %+v, want %+v", symbols, fixture.Expect.Symbols)}
	}
	if failure := compareGeneratedRelocations(fixture.ID, objectReport.RelocationRecords, fixture.Expect.RelocationRuns); failure != nil {
		return failure
	}
	return nil
}

func buildGeneratedObject(generator string, inputs []string) ([]byte, error) {
	switch generator {
	case "amd64-bigobj-v2":
		if len(inputs) != 1 {
			return nil, fmt.Errorf("amd64-bigobj-v2 requires exactly one input")
		}
		return pecoff.BuildAMD64BigObjectProbe(inputs[0])
	case "amd64-relocation-overflow":
		if len(inputs) != 1 {
			return nil, fmt.Errorf("amd64-relocation-overflow requires exactly one input")
		}
		return pecoff.BuildAMD64RelocationOverflowProbeObject(inputs[0])
	case "amd64-weak-external-alias":
		if len(inputs) != 6 {
			return nil, fmt.Errorf("amd64-weak-external-alias requires exactly six inputs")
		}
		return pecoff.BuildAMD64WeakExternalProbeObject(inputs[0], inputs[1], inputs[2], inputs[3], inputs[4], inputs[5])
	case "amd64-weak-search-modes":
		if len(inputs) != 6 {
			return nil, fmt.Errorf("amd64-weak-search-modes requires exactly six inputs")
		}
		return pecoff.BuildAMD64WeakSearchProbeObject(inputs[0], inputs[1], inputs[2], inputs[3], inputs[4], inputs[5])
	default:
		return nil, fmt.Errorf("unknown generator %q", generator)
	}
}

func compareGeneratedRelocations(caseID string, records []pecoff.ObjectRelocation, runs []generatedRelocationExpectation) *Failure {
	next := 0
	for _, run := range runs {
		if run.Start != next || run.Count <= 0 || run.Count > len(records)-next {
			return &Failure{CaseID: caseID, Detail: fmt.Sprintf("relocation run [%d,%d) is not a contiguous in-range partition", run.Start, run.Start+run.Count)}
		}
		for index := run.Start; index < run.Start+run.Count; index++ {
			record := records[index]
			if record.SectionIndex != run.SectionIndex || record.Section != run.Section || record.VirtualAddress != run.VirtualAddress ||
				record.SymbolIndex != run.SymbolIndex || record.Symbol != run.Symbol || record.Type != run.Type {
				return &Failure{CaseID: caseID, Detail: fmt.Sprintf("relocation %d mismatch: got %+v, want run %+v", index, record, run)}
			}
		}
		next += run.Count
	}
	if next != len(records) {
		return &Failure{CaseID: caseID, Detail: fmt.Sprintf("relocation runs cover %d records, want %d", next, len(records))}
	}
	return nil
}

func loadGeneratedManifest(path string) (generatedManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return generatedManifest{}, fmt.Errorf("open generated COFF object fixture manifest: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return generatedManifest{}, fmt.Errorf("inspect generated COFF object fixture manifest: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxManifestBytes {
		return generatedManifest{}, fmt.Errorf("generated COFF object fixture manifest has invalid size %d", info.Size())
	}
	contents, err := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	if err != nil {
		return generatedManifest{}, fmt.Errorf("read generated COFF object fixture manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var value generatedManifest
	if err := decoder.Decode(&value); err != nil {
		return generatedManifest{}, fmt.Errorf("decode generated COFF object fixture manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return generatedManifest{}, fmt.Errorf("decode generated COFF object fixture manifest trailing data")
	}
	if err := validateGeneratedRequiredFields(contents); err != nil {
		return generatedManifest{}, err
	}
	if value.Schema != "./schema-v0.json" || value.SchemaVersion != generatedFixtureSchema || value.Target != fixtureTarget {
		return generatedManifest{}, fmt.Errorf("generated COFF object fixture manifest identity is incompatible")
	}
	if len(value.Contracts) == 0 || len(value.Deferred) == 0 || len(value.Cases) == 0 {
		return generatedManifest{}, fmt.Errorf("generated COFF object fixture manifest is missing contracts, deferred coverage, or cases")
	}
	if err := validateStringList("contract", value.Contracts); err != nil {
		return generatedManifest{}, err
	}
	if err := validateStringList("deferred coverage", value.Deferred); err != nil {
		return generatedManifest{}, err
	}
	for _, fixture := range value.Cases {
		if fixture.Generator == "" || len(fixture.Inputs) == 0 || fixture.Expect.ByteLength <= 0 || !validSHA256(fixture.Expect.SHA256) ||
			len(fixture.Expect.Sections) == 0 || len(fixture.Expect.Symbols) == 0 || len(fixture.Expect.RelocationRuns) == 0 {
			return generatedManifest{}, fmt.Errorf("generated COFF object fixture %q has incomplete generator or expectation data", fixture.ID)
		}
	}
	return value, nil
}

func validateGeneratedRequiredFields(contents []byte) error {
	root, err := requiredJSONObject(json.RawMessage(contents), "generated root",
		"$schema", "schemaVersion", "target", "contracts", "deferred", "cases")
	if err != nil {
		return err
	}
	cases, err := requiredJSONArray(root["cases"], "generated cases")
	if err != nil {
		return err
	}
	for caseIndex, caseRaw := range cases {
		caseContext := fmt.Sprintf("generated cases[%d]", caseIndex)
		fixture, err := requiredJSONObject(caseRaw, caseContext, "id", "generator", "inputs", "expect")
		if err != nil {
			return err
		}
		expect, err := requiredJSONObject(fixture["expect"], caseContext+".expect",
			"byteLength", "sha256", "machine", "characteristics", "bigObject", "bigObjectVersion", "sections", "symbols", "relocationRuns")
		if err != nil {
			return err
		}
		for _, group := range []struct {
			name string
			keys []string
		}{
			{name: "sections", keys: []string{"name", "relocations", "characteristics", "comdatSelection", "associativeSection"}},
			{name: "symbols", keys: []string{"tableIndex", "name", "value", "sectionNumber", "type", "storageClass", "auxiliaryRecords", "weakFallbackIndex", "weakFallback", "weakSearch"}},
			{name: "relocationRuns", keys: []string{"start", "count", "sectionIndex", "section", "virtualAddress", "symbolIndex", "symbol", "type"}},
		} {
			items, err := requiredJSONArray(expect[group.name], caseContext+".expect."+group.name)
			if err != nil {
				return err
			}
			for itemIndex, item := range items {
				context := fmt.Sprintf("%s.expect.%s[%d]", caseContext, group.name, itemIndex)
				if _, err := requiredJSONObject(item, context, group.keys...); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
