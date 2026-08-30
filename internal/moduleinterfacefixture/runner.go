package moduleinterfacefixture

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	schemaVersion = "fairyc.module-interface-fixtures/v5"
	maximumSource = 1 << 20
)

// Run loads, compiles, extracts, and compares every interface invalidation case.
func Run(path string) (Report, error) {
	absoluteManifest, err := filepath.Abs(path)
	if err != nil {
		return Report{}, fmt.Errorf("resolve module interface manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	if definition.Schema == "" || definition.SchemaVersion != schemaVersion || len(definition.Contracts) == 0 || len(definition.Cases) == 0 ||
		definition.Identity.StateLayout.ContractUID == "" || len(definition.Identity.StateLayout.Fields) == 0 {
		return Report{}, fmt.Errorf("invalid module interface fixture contract header")
	}
	coordinateID, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{
		PackageUID:      definition.Identity.PackageUID,
		VersionIdentity: definition.Identity.VersionIdentity,
		Features:        definition.Identity.Features,
	})
	if err != nil {
		return Report{}, fmt.Errorf("invalid module interface fixture package identity: %w", err)
	}

	report := Report{CasesRun: len(definition.Cases)}
	hashes := make(map[string]moduleinterface.Hash, len(definition.Cases))
	seen := make(map[string]struct{}, len(definition.Cases))
	manifestDirectory := filepath.Dir(absoluteManifest)
	for _, fixture := range definition.Cases {
		if fixture.ID == "" {
			return Report{}, fmt.Errorf("module interface fixture has an empty case id")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return Report{}, fmt.Errorf("duplicate module interface fixture %q", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		hash, failure := runCase(manifestDirectory, fixture, coordinateID, definition.Identity.StateLayout)
		if failure != nil {
			report.Failures = append(report.Failures, *failure)
			continue
		}
		hashes[fixture.ID] = hash
	}
	for _, fixture := range definition.Cases {
		actual, found := hashes[fixture.ID]
		if !found {
			continue
		}
		switch fixture.Relation {
		case "baseline":
			if fixture.CompareTo != "" || fixture.ExpectHash == "" {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "baseline requires expectHash and no compareTo"})
				continue
			}
			if actual.String() != fixture.ExpectHash {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("hash=%q want=%q", actual, fixture.ExpectHash)})
			}
		case "same", "different":
			reference, referenceFound := hashes[fixture.CompareTo]
			if fixture.CompareTo == "" || !referenceFound {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("comparison fixture %q is unavailable", fixture.CompareTo)})
				continue
			}
			equal := actual == reference
			if fixture.Relation == "same" && !equal || fixture.Relation == "different" && equal {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("hash=%s compare=%s relation=%s", actual, reference, fixture.Relation)})
			}
		default:
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unknown relation %q", fixture.Relation)})
		}
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open module interface manifest: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode module interface manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode module interface manifest: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode module interface manifest: %w", err)
	}
	return definition, nil
}

func runCase(root string, fixture caseDefinition, coordinateID packagegraph.CoordinateID, layoutDefinition stateLayoutDefinition) (moduleinterface.Hash, *Failure) {
	path, err := containedSource(root, fixture.Source)
	if err != nil {
		return moduleinterface.Hash{}, &Failure{CaseID: fixture.ID, Detail: err.Error()}
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximumSource {
		return moduleinterface.Hash{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("source is missing, non-regular, or larger than %d bytes", maximumSource)}
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return moduleinterface.Hash{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("read source: %v", err)}
	}
	frontendResult, err := frontend.AnalyzeSyntax(source.NewFile("api.fy", body), source.DefaultLimits())
	if err != nil || len(frontendResult.Tree.Diagnostics) != 0 || frontendResult.AST == nil {
		return moduleinterface.Hash{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("syntax failed: error=%v diagnostics=%v", err, frontendResult.Tree.Diagnostics)}
	}
	semanticResult := semantic.Analyze([]semantic.File{{Path: "api.fy", Module: frontendResult.AST}})
	if semanticResult.CompletedThrough != semantic.PhaseFlow || semanticResult.FailedPhase != semantic.PhaseNone || len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return moduleinterface.Hash{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("semantic analysis failed: phase=%s diagnostics=%v unsupported=%v", semanticResult.FailedPhase, semanticResult.Diagnostics, semanticResult.Unsupported)}
	}
	stateFields := make([]hir.StateField, 0, len(layoutDefinition.Fields))
	for _, field := range layoutDefinition.Fields {
		stateFields = append(stateFields, hir.StateField{
			Name: field.Name, Kind: hir.StateFieldKind(field.Kind), KeyTypeID: field.KeyTypeID, ValueTypeID: field.ValueTypeID,
		})
	}
	semanticResult.HIR.StateLayouts = []hir.StateLayout{{
		Package: hir.DefaultPackage, ContractUID: layoutDefinition.ContractUID, Fields: stateFields,
	}}
	descriptor, err := moduleinterface.Extract(semanticResult.HIR, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: coordinateID})
	if err != nil {
		return moduleinterface.Hash{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("extract interface: %v", err)}
	}
	hash, err := moduleinterface.Compute(descriptor)
	if err != nil {
		return moduleinterface.Hash{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("hash interface: %v", err)}
	}
	return hash, nil
}

func containedSource(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" || filepath.Ext(relative) != ".fy" || strings.Contains(relative, "\\") {
		return "", fmt.Errorf("invalid source path %q", relative)
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source path %q escapes fixture root", relative)
	}
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve fixture root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, clean))
	if err != nil {
		return "", fmt.Errorf("resolve source path %q: %w", relative, err)
	}
	relativeResolved, err := filepath.Rel(rootResolved, resolved)
	if err != nil || relativeResolved == ".." || strings.HasPrefix(relativeResolved, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source path %q escapes fixture root", relative)
	}
	return resolved, nil
}
