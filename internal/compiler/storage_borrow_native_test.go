package compiler_test

import (
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func storageBorrowNativeMIR(t *testing.T, resource bool) mir.Program {
	t.Helper()
	source := storageBorrowNativeSource
	if resource {
		source = strings.NewReplacer("borrow_element<i32>", "borrow_element<Item>", "ref i32", "ref Item", "return *value;", "return value.value;").Replace(source)
	}
	analysis, err := analyzeStorageBorrow(t, source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !result.Verified {
		t.Fatalf("native borrow preparation: %v", err)
	}
	return result.MIR
}

func TestPrivateVectorStorageBorrowSourceBackendTypes(t *testing.T) {
	for _, resource := range []bool{false, true} {
		program := storageBorrowNativeMIR(t, resource)
		unit, err := cbackend.EmitStorageTransfers(program, hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
		if err != nil || !slices.Contains(unit.Features, "private-storage-borrow") {
			t.Fatalf("private borrow emission: %v", err)
		}
		changed := 0
		for fi := range program.Functions {
			for bi := range program.Functions[fi].Blocks {
				for oi := range program.Functions[fi].Blocks[bi].Operations {
					op := &program.Functions[fi].Blocks[bi].Operations[oi]
					if op.Kind == mir.OperationStorageBorrow {
						op.Operands[0], op.Operands[1] = op.Operands[1], op.Operands[0]
						changed++
					}
				}
			}
		}
		if changed != 1 {
			t.Fatalf("missing storage borrow: %d", changed)
		}
		if _, err := cbackend.EmitStorageTransfers(program, hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
			t.Fatal("swapped owner/address operands reached C emission")
		}
	}
}

func TestPrivateVectorStorageBorrowSourceNativeMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("storage-borrow execution requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range []bool{false, true} {
		name := "scalar"
		if resource {
			name = "resource"
		}
		t.Run(name, func(t *testing.T) {
			unit, err := cbackend.EmitStorageTransfers(storageBorrowNativeMIR(t, resource), hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			probe := storageBorrowNativeProbe
			if resource {
				probe = strings.Replace(probe, "return &item.value;", "return &item;", 1)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalar+probe)
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsHost(artifact)
					if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
						t.Fatalf("storage-borrow execution: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}

func mutableStorageBorrowNativeMIR(t *testing.T, resource bool) mir.Program {
	t.Helper()
	source := strings.NewReplacer(
		"fn ref T borrow_element<T>(ref Owner owner) { return unsafe { storage_borrow<T>(owner, owner.data) }; }",
		"fn ref mut T borrow_element<T>(ref mut Owner owner) { return unsafe { storage_borrow_mut<T>(owner, owner.data) }; }",
		"fn ref i32 relay(ref Owner owner) { return borrow_element<i32>(owner); }",
		"fn ref mut i32 relay(ref mut Owner owner) { return borrow_element<i32>(owner); }",
		"fn i32 read(ref i32 value)", "fn i32 read(ref mut i32 value)",
		"let Owner owner =", "var Owner owner =",
		"let ref i32 view = relay(ref owner);", "let ref mut i32 view = relay(ref mut owner);",
	).Replace(storageBorrowNativeSource)
	if resource {
		source = strings.NewReplacer(
			"borrow_element<i32>", "borrow_element<Item>",
			"ref mut i32", "ref mut Item",
			"return *value;", "value.value = 43; return value.value;",
		).Replace(source)
	}
	if source == storageBorrowNativeSource {
		t.Fatal("mutable native fixture unchanged")
	}
	analysis, err := analyzeStorageBorrow(t, source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !result.Verified {
		t.Fatalf("mutable native borrow preparation: %v", err)
	}
	return result.MIR
}

func TestPrivateVectorStorageBorrowSourceMutableBackendTypes(t *testing.T) {
	program := mutableStorageBorrowNativeMIR(t, false)
	entry := hir.QualifiedModulePath(1, "main") + "::main"
	if _, err := cbackend.Emit(program, entry, cbackend.Stage0WindowsMSVCProfile()); err == nil {
		t.Fatal("ordinary C emission admitted private mutable storage borrow")
	}
	unit, err := cbackend.EmitStorageTransfers(program, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil || !slices.Contains(unit.Features, "private-storage-borrow-mut") {
		t.Fatalf("private mutable borrow emission: %v", err)
	}
	changed := 0
	for fi := range program.Functions {
		for bi := range program.Functions[fi].Blocks {
			for oi := range program.Functions[fi].Blocks[bi].Operations {
				op := &program.Functions[fi].Blocks[bi].Operations[oi]
				if op.Kind == mir.OperationStorageBorrowMut {
					op.Operands[0], op.Operands[1] = op.Operands[1], op.Operands[0]
					changed++
				}
			}
		}
	}
	if changed != 1 {
		t.Fatalf("missing mutable storage borrow: %d", changed)
	}
	if _, err := cbackend.EmitStorageTransfers(program, entry, cbackend.Stage0WindowsMSVCProfile()); err == nil {
		t.Fatal("swapped mutable owner/address reached C emission")
	}
}

func TestPrivateVectorStorageBorrowSourceMutableNativeMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("mutable storage-borrow execution requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range []bool{false, true} {
		name := "scalar"
		if resource {
			name = "resource"
		}
		t.Run(name, func(t *testing.T) {
			unit, err := cbackend.EmitStorageTransfers(mutableStorageBorrowNativeMIR(t, resource), hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			probe := storageBorrowNativeProbe
			var exitCode int32 = 42
			if resource {
				probe = strings.NewReplacer(
					"return &item.value;", "return &item;",
					"owners == 1 && items == 0 && slots == 1", "owners == 1 && items == 0 && slots == 1 && item.value == 43",
				).Replace(probe)
				exitCode = 44
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalar+probe)
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsHost(artifact)
					if err != nil || outcome.ExitCode != exitCode || outcome.Stdout != "" || outcome.Stderr != "" {
						t.Fatalf("mutable storage-borrow execution: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}

const storageBorrowNativeSource = `module main;
@link_name("fairyc_borrow_slot") unsafe extern C fn ptr mut void slot();
@link_name("fairyc_borrow_owner_drop") unsafe extern C fn void owner_drop();
@link_name("fairyc_borrow_item_drop") unsafe extern C fn void item_drop();
@link_name("fairyc_borrow_finished") unsafe extern C fn i32 finished();
resource Item { i32 value; cleanup { unsafe { item_drop(); } } }
resource Owner { ptr mut void data; cleanup { unsafe { owner_drop(); } } }
fn ref T borrow_element<T>(ref Owner owner) { return unsafe { storage_borrow<T>(owner, owner.data) }; }
fn ref i32 relay(ref Owner owner) { return borrow_element<i32>(owner); }
fn i32 read(ref i32 value) { return *value; }
fn i32 exercise() {
    let Owner owner = Owner { .data = unsafe { slot() } };
    let ref i32 view = relay(ref owner);
    return read(view);
}
fn i32 main() { let i32 value = exercise(); return value + unsafe { finished() }; }
`

const storageBorrowNativeProbe = `
static struct { int32_t value; } item = { 41 };
static unsigned owners, items, slots;
void *fairyc_borrow_slot(void) { slots++; return &item.value; }
void fairyc_borrow_owner_drop(void) { owners++; }
void fairyc_borrow_item_drop(void) { items++; }
int32_t fairyc_borrow_finished(void) { return owners == 1 && items == 0 && slots == 1; }
`
