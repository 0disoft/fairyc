package workspacecompiler

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

const regionByteAllocationSource = `module main;
import std::alloc as alloc;
fn usize size_once(ref mut usize calls) { *calls = *calls + 1; return 4; }
fn i32 check(alloc::Arena input) {
    var alloc::Arena arena = input;
    var usize calls = 0;
    region scratch in ref mut arena {
        let i32 code = switch (scratch.alloc_zeroed_bytes(size_once(ref mut calls))) {
            case .ok(value): {
                assert(calls == 1 && value.len == 4 && value[0] == 0 && value[3] == 0);
                value[1] = 42;
                assert(value[1] == 42);
                42
            }
            case .err(_): { 1 }
        };
        assert(code == 42);
        assert(switch (scratch.alloc_zeroed_bytes(0)) {
            case .ok(empty): { empty.len == 0 && empty.is_empty }
            case .err(_): { false }
        });
    }
    return 42;
}
fn i32 main() { return switch (alloc::create()) { case .ok(value): { check(value) } case .err(_): { 1 } }; }
`

func TestRegionByteAllocationWorkspace(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	writeFile(t, filepath.Join(root, "src", "main.fy"), regionByteAllocationSource)
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
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	// This test loads the unmodified compiler-bundled SDK through the normal loader.
	options := interpreter.Options{AbstractArenaSDKPackages: []hir.PackageID{sdkOwner}}
	outcome, err := interpreter.ExecuteWithOptions(checked.MIR, entry, options)
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.RegionTrace) != 1 || outcome.RegionTrace[0].Placements != 2 {
		t.Fatalf("Region byte interpreter outcome: %+v %v", outcome, err)
	}
	// Reject corrupted selectors, argument types, and borrowed result payloads.
	mutations := []struct {
		name  string
		apply func(*mir.Operation)
	}{
		{"unknown-operator", func(op *mir.Operation) { op.Operator = "unknown" }},
		{"missing-operator", func(op *mir.Operation) { op.Operator = "" }},
		{"wrong-owner", func(op *mir.Operation) { op.OwnerType = hir.TypeI32 }},
		{"wrong-operand", func(op *mir.Operation) { op.Operands = []mir.Operand{{Kind: mir.OperandI32, I32: 4}} }},
		{"shared-payload", func(op *mir.Operation) {
			_, failure, _ := hir.ResultElements(op.Type)
			op.Type = hir.Result(hir.SharedBorrow(hir.TypeByteSlice), failure)
		}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			for fi := range checked.MIR.Functions {
				for bi := range checked.MIR.Functions[fi].Blocks {
					for oi := range checked.MIR.Functions[fi].Blocks[bi].Operations {
						op := &checked.MIR.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind != mir.OperationRegionPlace || op.Operator != "alloc_zeroed_bytes" {
							continue
						}
						original := *op
						mutation.apply(op)
						failures := mir.Verify(checked.MIR)
						*op = original
						if len(failures) == 0 {
							t.Fatal("malformed Region byte operation verified")
						}
						return
					}
				}
			}
			t.Fatal("missing Region byte operation")
		})
	}
}

func TestRegionByteAllocationRejectsEscapeAndWrongType(t *testing.T) {
	for _, item := range []struct{ name, source, code string }{
		{"escape", `module main; import std::alloc as alloc;
fn Result<ref mut u8[], alloc::AllocError> escape(alloc::Arena input) {
    var alloc::Arena arena = input;
    region scratch in ref mut arena { return scratch.alloc_zeroed_bytes(1); }
    return .err(alloc::AllocError { .status = 1 });
}
fn i32 main() { return 0; }`, "E3101"},
		{"exclusive-alias", `module main; import std::alloc as alloc;
fn void check(alloc::Arena input) { var alloc::Arena arena = input;
    region scratch in ref mut arena {
        discard switch (scratch.alloc_zeroed_bytes(2)) {
            case .ok(value): { let ref mut u8[] child = value[..]; value[0] = 7; assert(child[0] == 7); 0 }
            case .err(_): { 1 }
        };
    }
}
fn i32 main() { return 0; }`, "E2101"},
		{"wrong-type", `module main; import std::alloc as alloc;
fn void check(alloc::Arena input) { var alloc::Arena arena = input; region scratch in ref mut arena { discard scratch.alloc_zeroed_bytes(true); } }
fn i32 main() { return 0; }`, "E1006"},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := writeCacheWorkspace(t, testAnswer42)
			writeFile(t, filepath.Join(root, "src", "main.fy"), item.source)
			checked, err := Check(loadCacheWorkspace(t, root))
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified {
				t.Fatal("invalid Region byte source verified")
			}
			for _, diagnostic := range checked.Semantic.Diagnostics {
				if diagnostic.Code == item.code {
					return
				}
			}
			t.Fatalf("missing %s: diagnostics=%+v unsupported=%+v", item.code, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		})
	}
}
