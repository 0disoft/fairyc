package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspacecallcatalog"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type atlasCallWorkspaceDefinition struct {
	Name     string
	Packages int
	Calls    int
}

type preparedAtlasCallWorkspace struct {
	definition   atlasCallWorkspaceDefinition
	partitions   []atlasmap.Partition
	expectations map[string]atlasmap.LoadExpectation
	rootInstance string
	rootSymbol   string
	shape        buildmeasure.AtlasCallScaleShape
}

func measureAtlasCallScale(clockResolution int64) (buildmeasure.AtlasCallScaleReport, error) {
	temporaryRoot, err := os.MkdirTemp("", "fairyc-atlas-call-scale-")
	if err != nil {
		return buildmeasure.AtlasCallScaleReport{}, fmt.Errorf("create Atlas call scale root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		return buildmeasure.AtlasCallScaleReport{}, err
	}
	report := buildmeasure.AtlasCallScaleReport{
		Schema: buildmeasure.AtlasCallScaleSchema, Workload: "generated-star-workspace-call-shapes-v0",
		Scope: "workspace-call-index-construction-plus-outgoing-catalog-encode;workspace-generation-loading-compilation-map-planning-symbol-discovery-excluded",
		Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: clockResolution,
		AllocationMetric:   "runtime-memstats-totalalloc-and-mallocs-delta;process-wide;gc-uncontrolled",
		Setup:              "deterministic-generated-star-workspaces;prepare-and-warm-each-shape-before-sampling;fixed-partitions;first-observations-only",
		Decision:           "scaling-control-only-no-complexity-default-traversal-impact-or-regression-claim",
		IterationsPerShape: buildmeasure.AtlasCallScaleIterationsPerShape,
		CompilerBuild:      compilerBuild.String(), LogicalSchema: atlasmap.LogicalSchema,
		CatalogVersion: workspacecallcatalog.CatalogVersion, CatalogEncoding: workspacecallcatalog.CatalogEncoding,
		Environment: buildmeasure.AtlasCallScaleEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH},
	}
	for _, definition := range buildmeasure.AtlasCallScaleDefinitions() {
		shapeRoot := filepath.Join(temporaryRoot, definition.Name)
		if err := writeAtlasCallScaleWorkspace(shapeRoot, definition); err != nil {
			return buildmeasure.AtlasCallScaleReport{}, err
		}
		prepared, err := prepareAtlasCallWorkspace(shapeRoot, atlasCallWorkspaceDefinition{
			Name: definition.Name, Packages: definition.Packages, Calls: definition.Calls,
		}, compilerBuild)
		if err != nil {
			return buildmeasure.AtlasCallScaleReport{}, err
		}
		warmup, err := executeAtlasCallScale(prepared)
		if err != nil {
			return buildmeasure.AtlasCallScaleReport{}, fmt.Errorf("warm Atlas call scale shape %q: %w", definition.Name, err)
		}
		for sequence := 1; sequence <= buildmeasure.AtlasCallScaleIterationsPerShape; sequence++ {
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			started := time.Now()
			encoded, err := executeAtlasCallScale(prepared)
			duration := time.Since(started).Nanoseconds()
			runtime.ReadMemStats(&after)
			if err != nil {
				return buildmeasure.AtlasCallScaleReport{}, fmt.Errorf("measure Atlas call scale shape %q sample %d: %w", definition.Name, sequence, err)
			}
			if !bytes.Equal(encoded, warmup) {
				return buildmeasure.AtlasCallScaleReport{}, fmt.Errorf("Atlas call scale shape %q sample %d changed canonical bytes", definition.Name, sequence)
			}
			if after.TotalAlloc < before.TotalAlloc || after.Mallocs < before.Mallocs {
				return buildmeasure.AtlasCallScaleReport{}, fmt.Errorf("Atlas call scale process allocation counters moved backwards")
			}
			prepared.shape.Samples = append(prepared.shape.Samples, buildmeasure.AtlasCallScaleSample{
				Sequence: sequence, DurationNS: duration, BelowClockResolution: duration < clockResolution,
				TotalAllocBytes: after.TotalAlloc - before.TotalAlloc, Mallocs: after.Mallocs - before.Mallocs,
				OutputBytes: uint64(len(encoded)),
			})
		}
		report.Shapes = append(report.Shapes, prepared.shape)
	}
	if err := buildmeasure.ValidateAtlasCallScale(report); err != nil {
		return buildmeasure.AtlasCallScaleReport{}, err
	}
	return report, nil
}

func writeAtlasCallScaleWorkspace(root string, definition buildmeasure.AtlasCallScaleDefinition) error {
	if definition.Packages < 2 || definition.Calls < definition.Packages-1 {
		return fmt.Errorf("Atlas call scale definition %q is invalid", definition.Name)
	}
	libraryCount := definition.Packages - 1
	var manifest strings.Builder
	fmt.Fprintf(&manifest, "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"measurement.atlas-call-scale.%s\"\nname = \"%s\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/6\"\nsource = \"src\"\n[workspace]\nmembers = [\n", definition.Name, definition.Name)
	for index := 1; index <= libraryCount; index++ {
		fmt.Fprintf(&manifest, "  \"libs/p%03d\",\n", index)
	}
	manifest.WriteString("]\n")
	for index := 1; index <= libraryCount; index++ {
		fmt.Fprintf(&manifest, "[dependencies.p%03d]\npath = \"libs/p%03d\"\nversion = \"1.0.0\"\nfeatures = []\n", index, index)
	}
	if err := writeAtlasCallScaleFile(filepath.Join(root, "fairy.toml"), manifest.String()); err != nil {
		return err
	}
	var source strings.Builder
	source.WriteString("module main;\n")
	for index := 1; index <= libraryCount; index++ {
		fmt.Fprintf(&source, "import p%03d as p%03d;\n", index, index)
	}
	source.WriteString("fn i32 main() {\n    var i32 result = 0;\n")
	for call := 0; call < definition.Calls; call++ {
		library := call%libraryCount + 1
		fmt.Fprintf(&source, "    result += p%03d::value();\n", library)
	}
	source.WriteString("    return result;\n}\n")
	if err := writeAtlasCallScaleFile(filepath.Join(root, "src", "main.fy"), source.String()); err != nil {
		return err
	}
	for index := 1; index <= libraryCount; index++ {
		name := fmt.Sprintf("p%03d", index)
		memberManifest := fmt.Sprintf("schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"measurement.atlas-call-scale.%s.%s\"\nname = \"%s\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/6\"\nsource = \"src\"\n", definition.Name, name, name)
		if err := writeAtlasCallScaleFile(filepath.Join(root, "libs", name, "fairy.toml"), memberManifest); err != nil {
			return err
		}
		memberSource := fmt.Sprintf("module %s;\npub fn i32 value() { return 1; }\n", name)
		if err := writeAtlasCallScaleFile(filepath.Join(root, "libs", name, "src", name+".fy"), memberSource); err != nil {
			return err
		}
	}
	return nil
}

func writeAtlasCallScaleFile(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create Atlas call scale directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		return fmt.Errorf("write Atlas call scale file: %w", err)
	}
	return nil
}

func prepareAtlasCallWorkspace(
	root string,
	definition atlasCallWorkspaceDefinition,
	compilerBuild buildidentity.Identity,
) (preparedAtlasCallWorkspace, error) {
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		return preparedAtlasCallWorkspace{}, fmt.Errorf("load Atlas call workspace %q: %w", definition.Name, err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return preparedAtlasCallWorkspace{}, err
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return preparedAtlasCallWorkspace{}, err
	}
	if len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 || !checked.Verified {
		return preparedAtlasCallWorkspace{}, fmt.Errorf("Atlas call workspace %q did not produce clean verified MIR", definition.Name)
	}
	_, partitions, err := atlasmap.PrepareWorkspace(workspace, checked, compilerBuild)
	if err != nil {
		return preparedAtlasCallWorkspace{}, err
	}
	expectations, err := atlasmap.CurrentWorkspaceExpectations(workspace, checked, compilerBuild)
	if err != nil {
		return preparedAtlasCallWorkspace{}, err
	}
	if len(workspace.Packages) == 0 {
		return preparedAtlasCallWorkspace{}, fmt.Errorf("Atlas call workspace %q has no root package", definition.Name)
	}
	rootUID := workspace.Packages[0].Manifest.Package.UID
	rootInstance, rootSymbol := "", ""
	shape := buildmeasure.AtlasCallScaleShape{
		Name: definition.Name, Packages: len(partitions),
		Calls: definition.Calls, CrossPackageCalls: definition.Calls, SelectedCalls: definition.Calls,
		Policy: buildmeasure.AtlasCallProcessPolicy{
			MaxPackages: definition.Packages, MaxCalls: definition.Calls,
			MaxOutputBytes: buildmeasure.AtlasCallScaleMaxOutputBytes,
		},
	}
	for _, loaded := range workspace.Packages {
		for _, file := range loaded.Sources {
			shape.SourceFiles++
			if uint64(len(file.Bytes)) > ^uint64(0)-shape.SourceBytes {
				return preparedAtlasCallWorkspace{}, fmt.Errorf("Atlas call workspace source-byte count overflow")
			}
			shape.SourceBytes += uint64(len(file.Bytes))
		}
	}
	for _, partition := range partitions {
		header := partition.Header()
		for _, count := range []int{header.Counts.Files, header.Counts.Modules, header.Counts.Symbols, header.Counts.Functions, header.Counts.Relations, header.Counts.Completeness} {
			if count < 0 || uint64(count) > ^uint64(0)-shape.MapRecords {
				return preparedAtlasCallWorkspace{}, fmt.Errorf("Atlas call workspace map-record count overflow")
			}
			shape.MapRecords += uint64(count)
		}
		if header.PackageUID != rootUID {
			continue
		}
		rootInstance = header.PackageInstance
		for _, symbol := range partition.Symbols() {
			if symbol.Declaration.Module == "main" && symbol.Declaration.Kind == semanticsnapshot.DeclarationFunction && symbol.Declaration.Name == "main" {
				rootSymbol = symbol.Visible
			}
		}
	}
	if rootInstance == "" || rootSymbol == "" || shape.Packages != definition.Packages || shape.SourceFiles != definition.Packages {
		return preparedAtlasCallWorkspace{}, fmt.Errorf("Atlas call workspace %q has an incomplete root or package shape", definition.Name)
	}
	return preparedAtlasCallWorkspace{
		definition: definition, partitions: partitions, expectations: expectations,
		rootInstance: rootInstance, rootSymbol: rootSymbol, shape: shape,
	}, nil
}

func executeAtlasCallScale(prepared preparedAtlasCallWorkspace) ([]byte, error) {
	index, err := buildAtlasCallIndex(prepared)
	if err != nil {
		return nil, err
	}
	return workspacecallcatalog.Encode(
		index,
		prepared.rootInstance,
		prepared.rootSymbol,
		atlasmap.DirectionOutgoing,
		buildmeasure.AtlasCallScaleMaxOutputBytes,
	)
}

func buildAtlasCallIndex(prepared preparedAtlasCallWorkspace) (*atlasmap.WorkspaceCallIndex, error) {
	index, err := atlasmap.NewWorkspaceCallIndex(
		prepared.partitions,
		prepared.expectations,
		atlasmap.WorkspaceCallBudget{MaxPackages: prepared.definition.Packages, MaxCalls: prepared.definition.Calls},
	)
	if err != nil {
		return nil, err
	}
	usage := index.Usage()
	if usage.Packages != prepared.definition.Packages || usage.Calls != prepared.definition.Calls ||
		usage.CrossPackageCalls != prepared.definition.Calls {
		return nil, fmt.Errorf("Atlas call scale shape %q changed joined usage", prepared.definition.Name)
	}
	return index, nil
}
