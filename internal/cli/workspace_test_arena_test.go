package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const abstractArenaTests = `module tests::arena;
import std::alloc as alloc;
struct Pair { i32 value; }
resource Guard { i32 value; cleanup { assert(this.value == 42); } }
resource Holder { alloc::Arena arena; cleanup {} }
fn alloc::Arena make() {
    return switch (alloc::create()) {
        case .ok(arena): { arena }
        case .err(_): { assert(false); make() }
    };
}
fn Result<i32,i32> failure() { return .err(7); }
fn Result<i32,i32> route(bool propagate) {
    var alloc::Arena arena = make();
    region r in ref mut arena {
        let Guard guard = Guard { .value = 42 };
        assert(switch (r.place(42)) { case .ok(n): { *n == 42 } case .err(_): { false } });
        if (propagate) { let i32 n = failure()?; }
        return .ok(42);
    }
}
@test fn void a_create_place_and_reenter() {
    var alloc::Arena arena = make();
    var i32 count = 0;
    while (count < 3) {
        region r in ref mut arena {
            assert(switch (r.place(count)) { case .ok(n): { *n == count } case .err(_): { false } });
            count += 1;
            continue;
        }
    }
    region r in ref mut arena {
        var Pair original = Pair { .value = 42 };
        switch (r.place(original)) {
            case .ok(stored): { original.value = 0; assert(stored.value == 42); }
            case .err(_): { assert(false); }
        };
    }
    assert(count == 3);
}

@test fn void b_return_and_propagation() {
    assert(route(false).is_ok);
    assert(switch (route(true)) { case .ok(_): { false } case .err(n): { n == 7 } });
}
@test fn void c_nested_resource_cleanup() { let Holder owner = Holder { .arena = make() }; }
@test fn void d_trap_does_not_unwind() { let alloc::Arena arena = make(); assert(false); }
@test fn void e_fresh_test_after_trap() { let alloc::Arena arena = make(); }
`

func TestStableWorkspaceTestAbstractArena(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/arena.fy", abstractArenaTests)
	code, stdout, stderr := runForTest([]string{"test", "--json"}, root)
	var report struct {
		Count, Passed, Failed int
		Tests                 []struct {
			Identity, Outcome string
			TrapCategory      *string
		}
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil || code != ExitProgramTrap || stderr != "" || report.Count != 5 || report.Passed != 4 || report.Failed != 1 {
		t.Fatalf("abstract Arena tests = (%d, %s, %q): %+v %v", code, stdout, stderr, report, err)
	}
	if report.Tests[3].TrapCategory == nil || *report.Tests[3].TrapCategory != "assertion-failed" || report.Tests[4].Outcome != "passed" {
		t.Fatalf("trap isolation: %+v", report.Tests)
	}
	lock := filepath.Join(root, "fairy.lock")
	before, err := os.ReadFile(lock)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runForTest([]string{"test", "--locked", "--json", "--exact", "tests::arena::a_create_place_and_reenter"}, root)
	if err := json.Unmarshal([]byte(stdout), &report); err != nil || code != ExitSuccess || stderr != "" || report.Count != 1 || report.Passed != 1 {
		t.Fatalf("locked Arena test = (%d, %s, %q): %v", code, stdout, stderr, err)
	}
	after, err := os.ReadFile(lock)
	if err != nil || string(before) != string(after) {
		t.Fatalf("locked Arena test changed the lock: %v", err)
	}
	if entries, err := os.ReadDir(filepath.Join(root, "build")); !os.IsNotExist(err) && (err != nil || len(entries) != 0) {
		t.Fatalf("semantic test created native build artifacts: %v %v", entries, err)
	}
}

func TestStableWorkspaceTestAbstractArenaTraceAndAuthority(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/arena.fy", abstractArenaTests)
	workspace, _, err := loadWorkspaceWithLock(root, false)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err = workspaceloader.WithPackageTests(workspace, ".")
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("compile: %v %+v", err, checked.Semantic.Diagnostics)
	}
	options, err := workspaceArenaInterpreterOptions(workspace)
	if err != nil || len(options.AbstractArenaSDKPackages) != 1 {
		t.Fatalf("SDK owners: %+v %v", options, err)
	}
	owner, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	entry := hir.QualifiedModulePath(owner, "tests::arena") + "::b_return_and_propagation"
	outcome, err := interpreter.ExecuteVoidWithOptions(checked.MIR, entry, options)
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || len(outcome.DropTrace) != 4 || len(outcome.RegionTrace) != 2 {
		t.Fatalf("cleanup trace: %+v %v", outcome, err)
	}
	for i, drop := range outcome.DropTrace {
		want := hir.QualifiedModulePath(owner, "tests::arena") + "::Guard"
		if i%2 == 1 {
			want = hir.QualifiedModulePath(options.AbstractArenaSDKPackages[0], "std::alloc") + "::Arena"
		}
		if drop.ResourceType.String() != want {
			t.Fatalf("drop %d: %+v, want %s", i, drop, want)
		}
	}
	_, err = interpreter.ExecuteVoid(checked.MIR, entry)
	var capability interpreter.CapabilityError
	if !errors.As(err, &capability) {
		t.Fatalf("SDK names enabled an unconfigured adapter: %v", err)
	}
	trapEntry := hir.QualifiedModulePath(owner, "tests::arena") + "::d_trap_does_not_unwind"
	outcome, err = interpreter.ExecuteVoidWithOptions(checked.MIR, trapEntry, options)
	if err != nil || outcome.Kind != interpreter.OutcomeTrap || len(outcome.DropTrace) != 0 {
		t.Fatalf("trap cleanup: %+v %v", outcome, err)
	}
	for i := range workspace.Packages {
		for j := range workspace.Packages[i].Sources {
			file := &workspace.Packages[i].Sources[j]
			if file.Path == "std/alloc.fy" {
				file.Bytes = append(append([]byte(nil), file.Bytes...), '\n')
			}
		}
	}
	if _, err := workspaceArenaInterpreterOptions(workspace); err == nil {
		t.Fatal("modified SDK bytes were admitted")
	}
}

func TestStableWorkspaceTestAbstractArenaRejectsForeignCalls(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/foreign.fy", `module tests::foreign;
import std::alloc as alloc;
@link_name("fy_std_arena_initialize_v0")
unsafe extern C fn u32 fake(ptr mut void value);
@test fn void rejected() {
    discard alloc::create();
    let u32 status = unsafe { fake(null) };
    assert(status == 0);
}
`)
	code, stdout, stderr := runForTest([]string{"test", "--json"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "foreign ABI") {
		t.Fatalf("foreign call acquired Arena adapter authority: (%d, %q, %q)", code, stdout, stderr)
	}
}
