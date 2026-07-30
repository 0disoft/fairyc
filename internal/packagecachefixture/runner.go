package packagecachefixture

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const schemaVersion = "fairyc.package-cache-fixtures/v4"

// Run decodes and compares all whole-package cache-key fixtures.
func Run(path string) (Report, error) {
	definition, err := loadManifest(path)
	if err != nil {
		return Report{}, err
	}
	if definition.Schema == "" || definition.SchemaVersion != schemaVersion || len(definition.Contracts) == 0 || len(definition.Cases) == 0 {
		return Report{}, fmt.Errorf("invalid package cache fixture contract header")
	}
	report := Report{CasesRun: len(definition.Cases)}
	keys := make(map[string]packagecache.Key, len(definition.Cases))
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if fixture.ID == "" {
			return Report{}, fmt.Errorf("package cache fixture has an empty case id")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return Report{}, fmt.Errorf("duplicate package cache fixture %q", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		input, err := parseInput(applyOverride(definition.Base, fixture.Override))
		if err != nil {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: err.Error()})
			continue
		}
		key, err := packagecache.Compute(input)
		if err != nil {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("compute key: %v", err)})
			continue
		}
		keys[fixture.ID] = key
	}
	for _, fixture := range definition.Cases {
		actual, found := keys[fixture.ID]
		if !found {
			continue
		}
		switch fixture.Relation {
		case "baseline":
			if fixture.CompareTo != "" || fixture.ExpectKey == "" {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "baseline requires expectKey and no compareTo"})
			} else if actual.String() != fixture.ExpectKey {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("key=%q want=%q", actual, fixture.ExpectKey)})
			}
		case "same", "different":
			reference, ok := keys[fixture.CompareTo]
			if fixture.CompareTo == "" || !ok {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("comparison fixture %q is unavailable", fixture.CompareTo)})
				continue
			}
			equal := actual == reference
			if fixture.Relation == "same" && !equal || fixture.Relation == "different" && equal {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("key=%s compare=%s relation=%s", actual, reference, fixture.Relation)})
			}
		default:
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unknown relation %q", fixture.Relation)})
		}
	}
	return report, nil
}

func parseInput(value inputDefinition) (packagecache.Inputs, error) {
	standardLibrary, err := packagegraph.ParseInstanceID(value.StandardLibrary)
	if err != nil {
		return packagecache.Inputs{}, fmt.Errorf("parse standardLibrary: %w", err)
	}
	packageID, err := packagegraph.ParseInstanceID(value.Package)
	if err != nil {
		return packagecache.Inputs{}, fmt.Errorf("parse package: %w", err)
	}
	source, err := packagecontent.ParseDigest(value.Source)
	if err != nil {
		return packagecache.Inputs{}, fmt.Errorf("parse source: %w", err)
	}
	dependencies := make([]packagecache.Dependency, 0, len(value.Dependencies))
	for index, definition := range value.Dependencies {
		coordinate, err := packagegraph.ParseCoordinateID(definition.Coordinate)
		if err != nil {
			return packagecache.Inputs{}, fmt.Errorf("parse dependency %d coordinate: %w", index, err)
		}
		interfaceHash, err := moduleinterface.ParseHash(definition.Interface)
		if err != nil {
			return packagecache.Inputs{}, fmt.Errorf("parse dependency %d interface: %w", index, err)
		}
		dependencies = append(dependencies, packagecache.Dependency{Coordinate: coordinate, Interface: interfaceHash})
	}
	return packagecache.Inputs{
		CompilerBuild:        value.CompilerBuild,
		LanguageRevision:     value.LanguageRevision,
		MIRVersion:           value.MIRVersion,
		TargetSchema:         value.TargetSchema,
		ABIProfile:           value.ABIProfile,
		ToolchainFingerprint: value.ToolchainFingerprint,
		Backend:              value.Backend,
		ProfileFlags:         slices.Clone(value.ProfileFlags),
		RuntimeABI:           value.RuntimeABI,
		StandardLibrary:      standardLibrary,
		Package:              packageID,
		Source:               source,
		Dependencies:         dependencies,
		Features:             slices.Clone(value.Features),
	}, nil
}

func applyOverride(base inputDefinition, override inputOverride) inputDefinition {
	result := base
	result.ProfileFlags = slices.Clone(base.ProfileFlags)
	result.Dependencies = slices.Clone(base.Dependencies)
	result.Features = slices.Clone(base.Features)
	assignString(&result.CompilerBuild, override.CompilerBuild)
	assignString(&result.LanguageRevision, override.LanguageRevision)
	assignString(&result.MIRVersion, override.MIRVersion)
	assignString(&result.TargetSchema, override.TargetSchema)
	assignString(&result.ABIProfile, override.ABIProfile)
	assignString(&result.ToolchainFingerprint, override.ToolchainFingerprint)
	assignString(&result.Backend, override.Backend)
	assignString(&result.RuntimeABI, override.RuntimeABI)
	assignString(&result.StandardLibrary, override.StandardLibrary)
	assignString(&result.Package, override.Package)
	assignString(&result.Source, override.Source)
	if override.ProfileFlags != nil {
		result.ProfileFlags = slices.Clone(*override.ProfileFlags)
	}
	if override.Dependencies != nil {
		result.Dependencies = slices.Clone(*override.Dependencies)
	}
	if override.Features != nil {
		result.Features = slices.Clone(*override.Features)
	}
	return result
}

func assignString(target *string, value *string) {
	if value != nil {
		*target = *value
	}
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open package cache manifest: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode package cache manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode package cache manifest: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode package cache manifest: %w", err)
	}
	return definition, nil
}
