// Package pecofffixture executes the versioned AMD64 COFF object fixture contract.
package pecofffixture

import (
	"bytes"
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

const (
	fixtureSchema    = "fairyc.coff-object-fixtures/v0"
	fixtureTarget    = "x86_64-windows-msvc"
	maxManifestBytes = 1024 * 1024
)

type Failure struct {
	CaseID string
	Detail string
}

type Report struct {
	Target   string
	CasesRun int
	Passed   int
	Failures []Failure
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
	ObjectHex string      `json:"objectHex"`
	Expect    expectation `json:"expect"`
}

type expectation struct {
	Sections    []sectionExpectation    `json:"sections"`
	Symbols     []symbolExpectation     `json:"symbols"`
	Relocations []relocationExpectation `json:"relocations"`
}

type sectionExpectation struct {
	Name               string `json:"name"`
	Relocations        int    `json:"relocations"`
	Characteristics    uint32 `json:"characteristics"`
	COMDATSelection    uint8  `json:"comdatSelection"`
	AssociativeSection int    `json:"associativeSection"`
}

type symbolExpectation struct {
	TableIndex       uint32 `json:"tableIndex"`
	Name             string `json:"name"`
	Value            uint32 `json:"value"`
	SectionNumber    int16  `json:"sectionNumber"`
	Type             uint16 `json:"type"`
	StorageClass     uint8  `json:"storageClass"`
	AuxiliaryRecords uint8  `json:"auxiliaryRecords"`
}

type relocationExpectation struct {
	Section        string `json:"section"`
	VirtualAddress uint32 `json:"virtualAddress"`
	SymbolIndex    uint32 `json:"symbolIndex"`
	Symbol         string `json:"symbol"`
	Type           uint16 `json:"type"`
}

func Run(manifestPath string) (Report, error) {
	definition, err := loadManifest(manifestPath)
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
		if failure := executeCase(fixture); failure != nil {
			report.Failures = append(report.Failures, *failure)
			continue
		}
		report.Passed++
	}
	return report, nil
}

func executeCase(fixture fixtureCase) *Failure {
	if !validLowerHex(fixture.ObjectHex) {
		return &Failure{CaseID: fixture.ID, Detail: "object hex must be nonempty lowercase hexadecimal with complete bytes"}
	}
	object, err := hex.DecodeString(fixture.ObjectHex)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("decode object hex: %v", err)}
	}
	report, err := pecoff.InspectObject(object)
	if err != nil {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("inspect object: %v", err)}
	}
	sections := make([]sectionExpectation, len(report.Sections))
	for index, section := range report.Sections {
		sections[index] = sectionExpectation{
			Name: section.Name, Relocations: section.Relocations, Characteristics: section.Characteristics,
			COMDATSelection: section.COMDATSelection, AssociativeSection: section.AssociativeSection,
		}
	}
	symbols := make([]symbolExpectation, len(report.SymbolRecords))
	for index, symbol := range report.SymbolRecords {
		symbols[index] = symbolExpectation{
			TableIndex: symbol.TableIndex, Name: symbol.Name, Value: symbol.Value,
			SectionNumber: symbol.SectionNumber, Type: symbol.Type, StorageClass: symbol.StorageClass,
			AuxiliaryRecords: symbol.AuxiliaryRecords,
		}
	}
	relocations := make([]relocationExpectation, len(report.RelocationRecords))
	for index, relocation := range report.RelocationRecords {
		relocations[index] = relocationExpectation{
			Section: relocation.Section, VirtualAddress: relocation.VirtualAddress, SymbolIndex: relocation.SymbolIndex,
			Symbol: relocation.Symbol, Type: relocation.Type,
		}
	}
	if !reflect.DeepEqual(sections, fixture.Expect.Sections) {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("section receipt mismatch: got %+v, want %+v", sections, fixture.Expect.Sections)}
	}
	if !reflect.DeepEqual(symbols, fixture.Expect.Symbols) {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("symbol receipt mismatch: got %+v, want %+v", symbols, fixture.Expect.Symbols)}
	}
	if !reflect.DeepEqual(relocations, fixture.Expect.Relocations) {
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("relocation receipt mismatch: got %+v, want %+v", relocations, fixture.Expect.Relocations)}
	}
	return nil
}

func validCaseID(value string) bool {
	if !strings.HasPrefix(value, "pass.") || len(value) == len("pass.") {
		return false
	}
	for _, character := range value[len("pass."):] {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}

func validLowerHex(value string) bool {
	if len(value) == 0 || len(value)%2 != 0 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open COFF object fixture manifest: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return manifest{}, fmt.Errorf("inspect COFF object fixture manifest: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxManifestBytes {
		return manifest{}, fmt.Errorf("COFF object fixture manifest has invalid size %d", info.Size())
	}
	contents, err := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	if err != nil {
		return manifest{}, fmt.Errorf("read COFF object fixture manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var value manifest
	if err := decoder.Decode(&value); err != nil {
		return manifest{}, fmt.Errorf("decode COFF object fixture manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return manifest{}, fmt.Errorf("decode COFF object fixture manifest trailing data")
	}
	if err := validateRequiredManifestFields(contents); err != nil {
		return manifest{}, err
	}
	if value.Schema != "./schema-v0.json" || value.SchemaVersion != fixtureSchema || value.Target != fixtureTarget {
		return manifest{}, fmt.Errorf("COFF object fixture manifest identity is incompatible")
	}
	if len(value.Contracts) == 0 || len(value.Deferred) == 0 || len(value.Cases) == 0 {
		return manifest{}, fmt.Errorf("COFF object fixture manifest is missing contracts, deferred coverage, or cases")
	}
	if err := validateStringList("contract", value.Contracts); err != nil {
		return manifest{}, err
	}
	if err := validateStringList("deferred coverage", value.Deferred); err != nil {
		return manifest{}, err
	}
	return value, nil
}

func validateRequiredManifestFields(contents []byte) error {
	root, err := requiredJSONObject(json.RawMessage(contents), "root",
		"$schema", "schemaVersion", "target", "contracts", "deferred", "cases")
	if err != nil {
		return err
	}
	cases, err := requiredJSONArray(root["cases"], "cases")
	if err != nil {
		return err
	}
	for caseIndex, caseRaw := range cases {
		caseContext := fmt.Sprintf("cases[%d]", caseIndex)
		fixture, err := requiredJSONObject(caseRaw, caseContext, "id", "objectHex", "expect")
		if err != nil {
			return err
		}
		expect, err := requiredJSONObject(fixture["expect"], caseContext+".expect", "sections", "symbols", "relocations")
		if err != nil {
			return err
		}
		for _, group := range []struct {
			name string
			keys []string
		}{
			{name: "sections", keys: []string{"name", "relocations", "characteristics", "comdatSelection", "associativeSection"}},
			{name: "symbols", keys: []string{"tableIndex", "name", "value", "sectionNumber", "type", "storageClass", "auxiliaryRecords"}},
			{name: "relocations", keys: []string{"section", "virtualAddress", "symbolIndex", "symbol", "type"}},
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

func requiredJSONObject(raw json.RawMessage, context string, keys ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, fmt.Errorf("COFF object fixture manifest %s must be an object", context)
	}
	for _, key := range keys {
		value, found := object[key]
		if !found || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("COFF object fixture manifest %s is missing required field %q", context, key)
		}
	}
	return object, nil
}

func requiredJSONArray(raw json.RawMessage, context string) ([]json.RawMessage, error) {
	var values []json.RawMessage
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &values) != nil || values == nil {
		return nil, fmt.Errorf("COFF object fixture manifest %s must be an array", context)
	}
	return values, nil
}

func validateStringList(name string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" || value != strings.TrimSpace(value) {
			return fmt.Errorf("COFF object fixture manifest %s contains an empty or padded value", name)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("COFF object fixture manifest %s %q is repeated", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}
