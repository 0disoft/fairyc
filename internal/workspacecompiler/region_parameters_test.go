package workspacecompiler

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

const regionParameterSource = `module main;
import std::alloc as alloc;
fn Result<ref i32, alloc::AllocError> store(Region scratch, i32 value) {
    return scratch.place(value);
}
fn Result<ref i32, alloc::AllocError> forward(Region scratch, i32 value) {
    let ref i32 stored = store(scratch, value)?;
    return .ok(stored);
}
fn i32 read(Region scratch, i32 value) {
    return switch (store(scratch, value)) { case .ok(item): { *item } case .err(_): { 0 } };
}
fn i32 both(Region left, Region right) { return read(left, 20) + read(right, 22); }
fn i32 check(alloc::Arena input) {
    var alloc::Arena arena = input;
    region scratch in ref mut arena {
        let i32 answer = switch (forward(scratch, 42)) {
            case .ok(item): { *item }
            case .err(_): { 0 }
        };
        assert(answer == 42);
        assert(both(scratch, scratch) == 42);
    }
    region again in ref mut arena { assert(read(again, 42) == 42); }
    return 42;
}
fn i32 main() { return switch (alloc::create()) { case .ok(value): { check(value) } case .err(_): { 1 } }; }
`

const regionProviderSource = `module answer;
import std::alloc as alloc;
pub fn Result<ref i32, alloc::AllocError> store(Region input, i32 value) { return input.place(value); }
`

func regionProviderCaller(input string) string {
	return strings.NewReplacer("module main;", "module main;\nimport answer as provider;",
		"return scratch.place(value);", "return provider::store(scratch, value);").Replace(input)
}

func TestRegionParametersKeepCallerLifetime(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	writeFile(t, filepath.Join(root, "src", "main.fy"), regionParameterSource)
	workspace := loadCacheWorkspace(t, root)
	checked, err := Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("check: diagnostics=%+v unsupported=%+v error=%v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported, err)
	}
	rootPackage, _ := workspace.RootPackageID()
	sdkOwner, err := workspace.PackageID(sdkstdlib.PackagePath)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.ExecuteWithOptions(checked.MIR, hir.QualifiedModulePath(rootPackage, "main")+"::main",
		interpreter.Options{AbstractArenaSDKPackages: []hir.PackageID{sdkOwner}})
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 ||
		len(outcome.RegionTrace) != 2 || outcome.RegionTrace[0].Placements != 3 || outcome.RegionTrace[1].Placements != 1 {
		t.Fatalf("caller-owned Region: %+v %v", outcome, err)
	}
	for fi := range checked.MIR.Functions {
		for bi := range checked.MIR.Functions[fi].Blocks {
			call := &checked.MIR.Functions[fi].Blocks[bi].Terminator
			for ai, argument := range call.Arguments {
				if argument.Kind != mir.OperandRegion {
					continue
				}
				for _, bad := range []mir.Operand{
					{Kind: mir.OperandLocal, Local: argument.Local},
					{Kind: mir.OperandRegion, Local: 100000, Type: hir.TypeRegion},
				} {
					call.Arguments[ai] = bad
					failures := mir.Verify(checked.MIR)
					call.Arguments[ai] = argument
					found := false
					for _, failure := range failures {
						found = found || failure.Code == "V1403"
					}
					if !found {
						t.Fatalf("invalid Region argument accepted: %+v", failures)
					}
				}
			}
		}
	}
	for index := range checked.MIR.Functions {
		function := &checked.MIR.Functions[index]
		if !strings.HasSuffix(function.Name, "::store") {
			continue
		}
		parameter := function.Parameters[0]
		for _, operation := range []mir.Operation{
			{Kind: mir.OperationRegionReset, Symbol: hir.SymbolID(parameter), DestinationLocal: mir.NoLocal, ExitKind: "return"},
			{Kind: mir.OperationAssign, DestinationLocal: 100000, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: parameter}}},
		} {
			original := function.Blocks[0].Operations
			function.Blocks[0].Operations = append(append([]mir.Operation{}, original...), operation)
			failures := mir.Verify(checked.MIR)
			function.Blocks[0].Operations = original
			found := false
			for _, failure := range failures {
				found = found || failure.Code == "V1403"
			}
			if !found {
				t.Fatalf("Region %s mutation accepted: %+v", operation.Kind, failures)
			}
		}
		return
	}
	t.Fatal("missing store function")
}

func TestRegionParametersPersistentCache(t *testing.T) {
	root := writeCacheWorkspace(t, regionProviderSource)
	writeFile(t, filepath.Join(root, "src", "main.fy"), regionProviderCaller(regionParameterSource))
	workspace := loadCacheWorkspace(t, root)
	owner, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	sdkOwner, err := workspace.PackageID(sdkstdlib.PackagePath)
	if err != nil {
		t.Fatal(err)
	}
	configuration := persistentTestConfiguration(t.TempDir(), "region-input-capabilities")
	for _, warm := range []bool{false, true} {
		session, err := NewSession(configuration)
		if err != nil {
			t.Fatal(err)
		}
		checked, err := session.Check(workspace)
		closeErr := session.Close()
		if err != nil || closeErr != nil || !checked.Compiler.Verified {
			t.Fatalf("warm=%v check=%v close=%v", warm, err, closeErr)
		}
		if warm && (checked.Cache.Outcome != OutcomeAllHit || checked.Cache.Lowered) {
			t.Fatalf("Region identities were not restored from disk: %+v", checked.Cache)
		}
		outcome, err := interpreter.ExecuteWithOptions(checked.Compiler.MIR, hir.QualifiedModulePath(owner, "main")+"::main",
			interpreter.Options{AbstractArenaSDKPackages: []hir.PackageID{sdkOwner}})
		if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 ||
			len(outcome.RegionTrace) != 2 || outcome.RegionTrace[0].Placements != 3 || outcome.RegionTrace[1].Placements != 1 {
			t.Fatalf("warm=%v Region identities: %+v %v", warm, outcome, err)
		}
	}
}
