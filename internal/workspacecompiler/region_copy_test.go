package workspacecompiler

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

const regionByteCopySource = `module main;
import std::alloc as alloc;
fn ref u8[] source_once(ref mut usize calls, ref u8[] source) { *calls = *calls + 1; return source; }
fn i32 check(alloc::Arena input) {
    var alloc::Arena arena = input;
    var usize calls = 0;
    region scratch in ref mut arena {
        var u8[4] original = [65, 0, 255, 90];
        let Result<ref mut u8[], alloc::AllocError> result = scratch.copy_bytes(source_once(ref mut calls, ref original)[1..3]);
        original[1] = 7;
        let i32 code = switch (result) {
            case .ok(value): {
                assert(calls == 1 && value.len == 2 && value[0] == 0 && value[1] == 255);
                value[0] = 9;
                assert(original[1] == 7);
                assert(switch (scratch.copy_bytes(value)) {
                    case .ok(next): { next[0] = 8; value[0] == 9 && next[1] == 255 }
                    case .err(_): { false }
                });
                42
            }
            case .err(_): { 1 }
        };
        assert(code == 42);
        let Result<ref mut u8[], alloc::AllocError> retained = {
            var u8[2] ephemeral = [8, 9];
            scratch.copy_bytes(ref ephemeral)
        };
        assert(switch (retained) {
            case .ok(value): { value[0] == 8 && value[1] == 9 }
            case .err(_): { false }
        });
        assert(switch (scratch.copy_bytes(b"")) {
            case .ok(empty): { empty.len == 0 && empty.is_empty }
            case .err(_): { false }
        });
        assert(switch (scratch.alloc_zeroed_bytes(1)) {
            case .ok(zero): { zero.len == 1 && zero[0] == 0 }
            case .err(_): { false }
        });
        assert(switch (scratch.place(42)) {
            case .ok(value): { *value == 42 }
            case .err(_): { false }
        });
    }
    return 42;
}
fn i32 main() { return switch (alloc::create()) { case .ok(value): { check(value) } case .err(_): { 1 } }; }
`

func TestRegionByteCopyPersistentWorkspace(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	writeFile(t, filepath.Join(root, "src", "main.fy"), regionByteCopySource)
	workspace := loadCacheWorkspace(t, root)
	owner, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	sdkOwner, err := workspace.PackageID(sdkstdlib.PackagePath)
	if err != nil {
		t.Fatal(err)
	}
	configuration := persistentTestConfiguration(t.TempDir(), "region-byte-operations")
	for _, warm := range []bool{false, true} {
		session, err := NewSession(configuration)
		if err != nil {
			t.Fatal(err)
		}
		checked, err := session.Check(workspace)
		closeErr := session.Close()
		if err != nil || closeErr != nil || !checked.Compiler.Verified {
			t.Fatalf("warm=%v diagnostics=%+v unsupported=%+v err=%v close=%v", warm, checked.Compiler.Semantic.Diagnostics, checked.Compiler.Semantic.Unsupported, err, closeErr)
		}
		if !warm {
			assertRegionByteMIRRejectsMutations(t, &checked.Compiler.MIR, "copy_bytes")
		}
		if warm && (checked.Cache.Outcome != OutcomeAllHit || checked.Cache.Lowered) {
			t.Fatalf("not reused from disk: %+v", checked.Cache)
		}
		outcome, err := interpreter.ExecuteWithOptions(checked.Compiler.MIR, hir.QualifiedModulePath(owner, "main")+"::main", interpreter.Options{AbstractArenaSDKPackages: []hir.PackageID{sdkOwner}})
		if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.RegionTrace) != 1 || outcome.RegionTrace[0].Placements != 6 {
			t.Fatalf("warm=%v: %+v %v", warm, outcome, err)
		}
	}
}
