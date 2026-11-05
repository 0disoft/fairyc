package cli

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func verifyCandidateCallsSourceCommand(t *testing.T, image string) {
	t.Helper()
	for _, fixture := range []struct {
		name, source, subject string
		version, minimum      int
		genericDependency     bool
	}{
		{"function-v1", atlasGenericProductionUsedSource, "main", 1, 1, false},
		{"function-v2", atlasGenericProductionUsedSource, "main", 2, 1, false},
		{"unused-template", `module main;pub fn T unused<T>(T value){return value;}fn i32 main(){return 0;}`, "unused", 1, 1, false},
		{"cleanup-specializations", `module main;import answer as a;resource Token<T>{T value;cleanup{let T observed=a::id<T>(this.value);}}fn i32 main(){let a=Token<i32>{.value=1};let b=Token<u8>{.value=2};return a::value();}`, "Token", 2, 2, true},
		{"owner-namespaces", atlasOwnerQualifiedNamespaceSource(true), "main", 2, 0, false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			var root string
			if fixture.source == atlasGenericProductionUsedSource {
				root = candidateAtlasGenericProductionWorkspace(t, fixture.source)
			} else {
				root = writeWorkspaceSessionFixture(t, fixture.source)
			}
			if fixture.genericDependency {
				writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", `module answer;pub fn i32 value(){return 42;}pub fn T id<T>(T value){return value;}`)
			}
			candidateCallsCommandSetup(t, root)
			f := candidateCallsCommandFixture(t, image, root, fixture.version)
			const uid = "example.session.app"
			instance, symbol := candidateCallsCommandSubject(t, f, uid, fixture.subject, true)
			budget := atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}
			golden, err := candidateCallsOracle(t, f, budget, instance, symbol, atlasmap.DirectionBoth, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			index, err := candidateCallsIndex(f, budget)
			if err != nil {
				t.Fatal(err)
			}
			if fixture.genericDependency {
				cleanups := 0
				for _, c := range index.Calls() {
					if c.SourceBody == "cleanup" && c.SourceSpecialization != "" && c.TargetSpecialization != "" {
						cleanups++
					}
				}
				if cleanups != 2 {
					t.Fatal("missing independent cross-package cleanup implementations")
				}
			}
			before := remoteTree(t, root)
			args := append(candidateCallsCommandArgs(fixture.version, uid, symbol, atlasmap.DirectionBoth, budget, 1<<20), "--locked")
			code, out, stderr := fetchCLIProcess(t, image, root, args, "")
			if code != 0 || out != string(golden) || stderr != "" {
				t.Fatalf("source calls: %d %.5000s %.500s; want %.5000s", code, out, stderr, golden)
			}
			for version := 0; version < fixture.minimum; version++ {
				args := append(candidateCallsCommandArgs(version, uid, symbol, atlasmap.DirectionBoth, budget, 1<<20), "--locked")
				if code, out, stderr := fetchCLIProcess(t, image, root, args, ""); code != 3 || out != "" || stderr == "" {
					t.Fatalf("unsupported source catalog %d: %d %q %q", version, code, out, stderr)
				}
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("source calls changed workspace")
			}
		})
	}
}

func verifyCandidateCallsSDKCommand(t *testing.T, image string) {
	t.Helper()
	for _, fixture := range []struct {
		name, source string
		version      int
	}{
		{"ordinary-sdk", atlasSDKCommandSource, 0},
		{"used-vector", candidateAtlasSDKVectorUsedSource, 2},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, fixture.source)
			candidateCallsCommandSetup(t, root)
			f := candidateCallsCommandFixture(t, image, root, fixture.version)
			const uid = "example.session.app"
			instance, symbol := candidateCallsCommandSubject(t, f, uid, "main", false)
			budget := atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}
			golden, err := candidateCallsOracle(t, f, budget, instance, symbol, atlasmap.DirectionBoth, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			before := remoteTree(t, root)
			args := append(candidateCallsCommandArgs(fixture.version, uid, symbol, atlasmap.DirectionBoth, budget, 1<<20), "--locked")
			code, out, stderr := candidateAtlasSDKCLIProcess(t, image, root, args)
			if code != 0 || out != string(golden) || stderr != "" {
				t.Fatalf("SDK calls: %d %.5000s %.500s; want %.5000s", code, out, stderr, golden)
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("SDK calls changed workspace")
			}
		})
	}
}
