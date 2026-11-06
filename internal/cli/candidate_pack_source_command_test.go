package cli

import (
	"reflect"
	"testing"
)

func verifyCandidatePackSourceCommand(t *testing.T, image string) {
	t.Helper()
	for _, fixture := range []struct {
		name, source, subject string
		version, minimum      int
		genericDependency     bool
	}{
		{"functions-v2", atlasGenericProductionUsedSource, "main", 2, 2, false},
		{"functions-v3", atlasGenericProductionUsedSource, "main", 3, 2, false},
		{"unused-template", `module main;pub fn T unused<T>(T value){return value;}fn i32 main(){return 0;}`, "unused", 2, 2, false},
		{"cleanup-specializations", `module main;import answer as a;resource Token<T>{T value;cleanup{let T observed=a::id<T>(this.value);}}fn i32 main(){let a=Token<i32>{.value=1};let b=Token<u8>{.value=2};return a::value();}`, "Token", 3, 3, true},
		{"owner-namespaces", atlasOwnerQualifiedNamespaceSource(true), "main", 3, 1, false},
		{"cleanup-local-drops", `module main;import answer as a;resource Token{i32 value;cleanup{}}resource Wrap<T>{Token inner;T marker;cleanup{let t=Token{.value=1};}}fn i32 main(){let t=Wrap<i32>{.inner=Token{.value=2},.marker=3};return 0;}`, "Wrap", 3, 3, false},
		{"move-sites", `module main;resource Token{i32 value;cleanup{}}fn Token choose(bool flag){return if(flag){let left=Token{.value=1};left}else{let right=Token{.value=2};right};}fn i32 main(){let token=choose(false);let other=token;return other.value;}`, "choose", 1, 1, false},
		{"cleanup-move-sites", `module main;resource Token{i32 value;cleanup{}}resource Wrap<T>{T marker;cleanup{let token=Token{.value=1};let other=token;}}fn i32 main(){let value=Wrap<i32>{.marker=1};return 0;}`, "Wrap", 3, 3, false},
		{"unsafe-sites", `module main;unsafe extern C fn i32 native_value(i32 n);fn i32 main(){let u8 n=1;discard unsafe{ref n as ptr u8};return unsafe{native_value(1)};}`, "main", 1, 1, false},
		{"cleanup-unsafe-sites", `module main;resource Token<T>{T value;cleanup{let u8 n=1;discard unsafe{ref n as ptr u8};}}fn i32 main(){let t=Token<i32>{.value=1};return 0;}`, "Token", 3, 3, false},
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
			verifyCandidatePackSourceFixture(t, image, root, fixture.subject, fixture.version, fixture.minimum)
		})
	}
	for _, fixture := range atlasReturnErrorProductionFixtures() {
		t.Run("errors-"+fixture.name, func(t *testing.T) {
			subject := "exercise"
			switch fixture.name {
			case "operand-order":
				subject = "before"
			case "generic":
				subject = "forward"
			}
			verifyCandidatePackSourceFixture(t, image, atlasReturnErrorProductionWorkspace(t, fixture), subject, fixture.minimum, fixture.minimum)
		})
	}
	for _, fixture := range atlasBorrowCollectorCases() {
		version, subject := 1, "exercise"
		switch fixture.name {
		case "shared-local", "mutable-local", "returned-parameter", "retained-call-result", "conditional-origins":
		case "generic":
			version = 2
		case "cleanup":
			version, subject = 3, "Box"
		default:
			continue
		}
		t.Run("borrows-"+fixture.name, func(t *testing.T) {
			verifyCandidatePackSourceFixture(t, image, writeWorkspaceSessionFixture(t, fixture.source), subject, version, version)
		})
	}
}

func verifyCandidatePackSourceFixture(t *testing.T, image, root, subject string, version, minimum int) {
	t.Helper()
	candidateCallsCommandSetup(t, root)
	f := candidatePackCommandFixture(t, image, root, version)
	q := candidatePackCommandControls(t, f, subject)
	q.Symbol = candidatePackCommandSubject(t, f, subject, true)
	before := remoteTree(t, root)
	observations := 0
	for _, action := range []string{"overview", "show", "around"} {
		checkCandidatePackCommand(t, image, root, f, q, action, false)
		observations++
	}
	for version := 1; version < minimum; version++ {
		q.Version = version
		observations++
		if code, out, stderr := fetchCLIProcess(t, image, root, candidatePackCommandArgs(f, q, "around"), ""); code != 3 || out != "" || stderr == "" {
			t.Fatalf("unsupported pack family v%d: %d %.500s %.500s", version, code, out, stderr)
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("source packs changed the workspace")
	}
	t.Logf("pack source command native processes: %d", observations)
}

func verifyCandidatePackSDKCommand(t *testing.T, image string) {
	t.Helper()
	type fixture struct {
		name, source, subject string
		version               int
	}
	fixtures := []fixture{
		{"ordinary-sdk", atlasSDKCommandSource, "main", 1},
		{"used-vector", candidateAtlasSDKVectorUsedSource, "main", 3},
	}
	for _, allocation := range atlasAllocationProductionFixtures() {
		subject := "exercise"
		switch allocation.name {
		case "operand-order":
			subject = "before"
		case "generic":
			subject = "put"
		case "cleanup":
			subject = "Wrap"
		}
		fixtures = append(fixtures, fixture{"allocations-" + allocation.name, allocation.source, subject, allocation.minimum})
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, fixture.source)
			candidateCallsCommandSetup(t, root)
			f := candidatePackCommandFixture(t, image, root, fixture.version)
			q := candidatePackCommandControls(t, f, fixture.subject)
			before := remoteTree(t, root)
			for _, action := range []string{"overview", "show", "around"} {
				checkCandidatePackCommand(t, image, root, f, q, action, true)
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("SDK packs changed the workspace")
			}
			t.Log("pack SDK command native processes: 3")
		})
	}
}
