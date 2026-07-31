package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/workspacecallcatalog"
)

const atlasCallTopologyMaxOutputBytes = uint64(2 << 20)

type atlasCallTopologyPackage struct {
	name         string
	dependencies []string
}

type atlasCallTopologyCheck struct {
	packageName string
	direction   atlasmap.QueryDirection
	selected    int
}

type atlasCallTopologyDefinition struct {
	name     string
	packages []atlasCallTopologyPackage
	checks   []atlasCallTopologyCheck
}

func TestAtlasCallTopologyCorpusJoinsNonStarWorkspaces(t *testing.T) {
	t.Parallel()
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	definitions := atlasCallTopologyDefinitions()
	for _, definition := range definitions {
		definition := definition
		t.Run(definition.name, func(t *testing.T) {
			t.Parallel()
			root := filepath.Join(t.TempDir(), definition.name)
			if err := writeAtlasCallTopologyWorkspace(root, definition); err != nil {
				t.Fatal(err)
			}
			callCount := atlasCallTopologyCallCount(definition)
			prepared, err := prepareAtlasCallWorkspace(root, atlasCallWorkspaceDefinition{
				Name: definition.name, Packages: len(definition.packages), Calls: callCount,
			}, compilerBuild)
			if err != nil {
				t.Fatal(err)
			}
			index, err := atlasmap.NewWorkspaceCallIndex(
				prepared.partitions,
				prepared.expectations,
				atlasmap.WorkspaceCallBudget{MaxPackages: len(definition.packages), MaxCalls: callCount},
			)
			if err != nil {
				t.Fatal(err)
			}
			if usage := index.Usage(); usage != (atlasmap.WorkspaceCallUsage{
				Packages: len(definition.packages), Calls: callCount, CrossPackageCalls: callCount,
			}) {
				t.Fatalf("topology usage = %+v", usage)
			}
			for _, check := range definition.checks {
				packageInstance, symbolID := atlasCallTopologySymbol(t, prepared, check.packageName)
				selected := atlasCallTopologySelected(t, index, packageInstance, symbolID, check.direction)
				if selected != check.selected {
					t.Fatalf("%s %s selected %d calls, want %d", check.packageName, check.direction, selected, check.selected)
				}
				first, err := workspacecallcatalog.Encode(index, packageInstance, symbolID, check.direction, atlasCallTopologyMaxOutputBytes)
				if err != nil {
					t.Fatal(err)
				}
				second, err := workspacecallcatalog.Encode(index, packageInstance, symbolID, check.direction, atlasCallTopologyMaxOutputBytes)
				if err != nil {
					t.Fatal(err)
				}
				if len(first) == 0 || !bytes.Equal(first, second) {
					t.Fatalf("%s %s catalog is not stable", check.packageName, check.direction)
				}
			}
		})
	}
}

func atlasCallTopologyDefinitions() []atlasCallTopologyDefinition {
	return []atlasCallTopologyDefinition{
		{
			name: "topology-chain",
			packages: []atlasCallTopologyPackage{
				{name: "root", dependencies: []string{"middle"}},
				{name: "middle", dependencies: []string{"leaf"}},
				{name: "leaf"},
			},
			checks: []atlasCallTopologyCheck{
				{packageName: "root", direction: atlasmap.DirectionOutgoing, selected: 1},
				{packageName: "middle", direction: atlasmap.DirectionBoth, selected: 2},
				{packageName: "leaf", direction: atlasmap.DirectionIncoming, selected: 1},
			},
		},
		{
			name: "topology-diamond",
			packages: []atlasCallTopologyPackage{
				{name: "root", dependencies: []string{"left", "right"}},
				{name: "left", dependencies: []string{"leaf"}},
				{name: "right", dependencies: []string{"leaf"}},
				{name: "leaf"},
			},
			checks: []atlasCallTopologyCheck{
				{packageName: "root", direction: atlasmap.DirectionOutgoing, selected: 2},
				{packageName: "left", direction: atlasmap.DirectionBoth, selected: 2},
				{packageName: "leaf", direction: atlasmap.DirectionIncoming, selected: 2},
			},
		},
		{
			name: "topology-fan-in",
			packages: []atlasCallTopologyPackage{
				{name: "root", dependencies: []string{"worker-a", "worker-b", "worker-c"}},
				{name: "worker-a", dependencies: []string{"sink"}},
				{name: "worker-b", dependencies: []string{"sink"}},
				{name: "worker-c", dependencies: []string{"sink"}},
				{name: "sink"},
			},
			checks: []atlasCallTopologyCheck{
				{packageName: "root", direction: atlasmap.DirectionOutgoing, selected: 3},
				{packageName: "worker-a", direction: atlasmap.DirectionBoth, selected: 2},
				{packageName: "sink", direction: atlasmap.DirectionIncoming, selected: 3},
			},
		},
	}
}

func atlasCallTopologyCallCount(definition atlasCallTopologyDefinition) int {
	result := 0
	for _, packageValue := range definition.packages {
		result += len(packageValue.dependencies)
	}
	return result
}

func writeAtlasCallTopologyWorkspace(root string, definition atlasCallTopologyDefinition) error {
	if len(definition.packages) < 3 || definition.packages[0].name != "root" {
		return fmt.Errorf("Atlas call topology %q has an invalid package order", definition.name)
	}
	var manifest strings.Builder
	fmt.Fprintf(&manifest, "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"measurement.atlas-call-topology.%s\"\nname = \"%s\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/7\"\nsource = \"src\"\n[workspace]\nmembers = [\n", definition.name, definition.name)
	members := make([]string, 0, len(definition.packages)-1)
	for _, packageValue := range definition.packages[1:] {
		members = append(members, "libs/"+packageValue.name)
	}
	slices.Sort(members)
	for _, member := range members {
		fmt.Fprintf(&manifest, "  \"%s\",\n", member)
	}
	manifest.WriteString("]\n")
	for _, dependency := range definition.packages[0].dependencies {
		fmt.Fprintf(&manifest, "[dependencies.%s]\npath = \"libs/%s\"\nversion = \"1.0.0\"\nfeatures = []\n", dependency, dependency)
	}
	if err := writeAtlasCallScaleFile(filepath.Join(root, "fairy.toml"), manifest.String()); err != nil {
		return err
	}
	if err := writeAtlasCallScaleFile(filepath.Join(root, "src", "main.fy"), atlasCallTopologySource("main", "main", definition.packages[0].dependencies)); err != nil {
		return err
	}
	for _, packageValue := range definition.packages[1:] {
		var memberManifest strings.Builder
		fmt.Fprintf(&memberManifest, "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"measurement.atlas-call-topology.%s.%s\"\nname = \"%s\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/7\"\nsource = \"src\"\n", definition.name, packageValue.name, packageValue.name)
		for _, dependency := range packageValue.dependencies {
			fmt.Fprintf(&memberManifest, "[dependencies.%s]\npath = \"libs/%s\"\nversion = \"1.0.0\"\nfeatures = []\n", dependency, dependency)
		}
		memberRoot := filepath.Join(root, "libs", packageValue.name)
		if err := writeAtlasCallScaleFile(filepath.Join(memberRoot, "fairy.toml"), memberManifest.String()); err != nil {
			return err
		}
		module := atlasCallTopologyModule(packageValue.name)
		if err := writeAtlasCallScaleFile(filepath.Join(memberRoot, "src", module+".fy"), atlasCallTopologySource(module, "value", packageValue.dependencies)); err != nil {
			return err
		}
	}
	return nil
}

func atlasCallTopologySource(module, function string, dependencies []string) string {
	var source strings.Builder
	fmt.Fprintf(&source, "module %s;\n", module)
	for _, dependency := range dependencies {
		module := atlasCallTopologyModule(dependency)
		fmt.Fprintf(&source, "import %s as %s;\n", module, module)
	}
	if function == "value" {
		source.WriteString("pub ")
	}
	fmt.Fprintf(&source, "fn i32 %s() {\n    return ", function)
	if len(dependencies) == 0 {
		source.WriteString("1")
	} else {
		for index, dependency := range dependencies {
			if index > 0 {
				source.WriteString(" + ")
			}
			fmt.Fprintf(&source, "%s::value()", atlasCallTopologyModule(dependency))
		}
	}
	source.WriteString(";\n}\n")
	return source.String()
}

func atlasCallTopologyModule(packageName string) string {
	return strings.ReplaceAll(packageName, "-", "_")
}

func atlasCallTopologySymbol(t *testing.T, prepared preparedAtlasCallWorkspace, packageName string) (string, string) {
	t.Helper()
	if packageName == "root" {
		return prepared.rootInstance, prepared.rootSymbol
	}
	expectedUID := "measurement.atlas-call-topology." + prepared.definition.Name + "." + packageName
	for _, partition := range prepared.partitions {
		header := partition.Header()
		if header.PackageUID != expectedUID {
			continue
		}
		for _, symbol := range partition.Symbols() {
			if symbol.Declaration.Module == atlasCallTopologyModule(packageName) && symbol.Declaration.Name == "value" {
				return header.PackageInstance, symbol.Visible
			}
		}
	}
	t.Fatalf("topology package %q has no value symbol", packageName)
	return "", ""
}

func atlasCallTopologySelected(
	t *testing.T,
	index *atlasmap.WorkspaceCallIndex,
	packageInstance, symbolID string,
	direction atlasmap.QueryDirection,
) int {
	t.Helper()
	result := 0
	if direction == atlasmap.DirectionOutgoing || direction == atlasmap.DirectionBoth {
		calls, err := index.OutgoingCalls(packageInstance, symbolID)
		if err != nil {
			t.Fatal(err)
		}
		result += len(calls)
	}
	if direction == atlasmap.DirectionIncoming || direction == atlasmap.DirectionBoth {
		calls, err := index.IncomingCalls(packageInstance, symbolID)
		if err != nil {
			t.Fatal(err)
		}
		result += len(calls)
	}
	return result
}
