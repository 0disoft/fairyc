package compiler_test

import (
	"encoding/json"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func storageTakeMIR(t *testing.T) mir.Program {
	t.Helper()
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
@link_name("fairyc_fixture_take_slot") unsafe extern C fn ptr mut void slot_raw();
@link_name("fairyc_fixture_take_drop") unsafe extern C fn u32 drop_raw(ptr mut void handle);
@link_name("fairyc_fixture_take_id") unsafe extern C fn i32 id_raw(ptr mut void handle);
@link_name("fairyc_fixture_take_finished") unsafe extern C fn i32 finished();
resource Item { ptr mut void handle; cleanup { let u32 status = unsafe { drop_raw(this.handle) }; } }
fn Item take(ptr mut void slot, i32 distractor) { return unsafe { Item { .handle = slot } }; }
fn Item relay(Item item) { return item; }
fn i32 exercise() {
    let ptr mut void slot = unsafe { slot_raw() };
    assert(slot != null);
    let Item value = take(slot, 7);
    let Item moved = relay(value);
    return unsafe { id_raw(moved.handle) };
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`)
	analysis := semantic.Analyze([]semantic.File{file})
	if len(analysis.Diagnostics) != 0 || len(analysis.Unsupported) != 0 {
		t.Fatalf("fixture semantics: %+v %+v", analysis.Diagnostics, analysis.Unsupported)
	}
	inserted := false
	for mi := range analysis.HIR.Modules {
		for fi := range analysis.HIR.Modules[mi].Functions {
			function := &analysis.HIR.Modules[mi].Functions[fi]
			if function.Name != "take" {
				continue
			}
			unsafe := function.Body.Statements[0].Expression
			if unsafe.Kind != "unsafe" || len(unsafe.Children) != 1 || unsafe.Children[0].Block == nil {
				t.Fatal("fixture unsafe block changed")
			}
			record := unsafe.Children[0].Block.Tail
			if record == nil || record.Kind != "record" || len(record.Children) != 1 {
				t.Fatal("fixture record changed")
			}
			pointer, span, typ, scope := record.Children[0], record.Span, record.Type, unsafe.Span
			*record = hir.Expression{Kind: "storage-take", Type: typ, TargetType: typ, Span: span,
				Symbol: hir.NoSymbol, UnsafeScope: &scope, Children: []hir.Expression{pointer}}
			inserted = true
		}
	}
	if !inserted {
		t.Fatal("private take was not inserted")
	}
	program, err := mir.Lower(analysis.HIR)
	if err != nil {
		t.Fatal(err)
	}
	if failures := mir.VerifyStorageTransfers(program); len(failures) != 0 {
		t.Fatal(failures)
	}
	return program
}

func TestPrivateVectorStorageTakeConsumerBoundaries(t *testing.T) {
	program := storageTakeMIR(t)
	if len(mir.Verify(program)) == 0 || len(mir.VerifyStorageQueries(program)) == 0 {
		t.Fatal("ordinary or query-only verifier admitted an unsafe take")
	}
	if _, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
		t.Fatal("ordinary C emitter admitted an unsafe take")
	}
	if _, err := interpreter.Execute(program, "main::main"); err == nil {
		t.Fatal("interpreter admitted an unsafe take")
	}
	if _, err := mircodec.Encode(program, 1<<20); err == nil {
		t.Fatal("artifact encoder admitted an unsafe take")
	}
	encoded, err := json.Marshal(map[string]any{"schema": mircodec.Schema, "mirVersion": mircodec.MIRVersion, "program": program})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mircodec.Decode(encoded, 1<<20); err == nil {
		t.Fatal("artifact decoder admitted an unsafe take")
	}
}

func TestPrivateVectorStorageTakeRejectsMalformedMIR(t *testing.T) {
	for name, mutate := range map[string]func(*mir.Operation){
		"unknown type": func(op *mir.Operation) { op.Type = "main::Missing" },
		"borrow":       func(op *mir.Operation) { op.Type = hir.SharedBorrow("main::Item") },
		"pointer":      func(op *mir.Operation) { op.OwnerType = hir.RawPointer(hir.TypeVoid, false) },
		"result":       func(op *mir.Operation) { op.DestinationLocal = mir.NoLocal },
		"operand":      func(op *mir.Operation) { op.Operands = nil },
		"null literal": func(op *mir.Operation) { op.Operands[0] = mir.Operand{Kind: mir.OperandNull} },
		"scope":        func(op *mir.Operation) { op.UnsafeScope = hir.Span{} },
		"entity":       func(op *mir.Operation) { op.Entity = "forged" },
		"move flag":    func(op *mir.Operation) { op.MovePayload = true },
		"field":        func(op *mir.Operation) { op.Field = "handle" },
	} {
		t.Run(name, func(t *testing.T) {
			program := storageTakeMIR(t)
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					for oi := range program.Functions[fi].Blocks[bi].Operations {
						op := &program.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind == mir.OperationStorageTake {
							mutate(op)
						}
					}
				}
			}
			if len(mir.VerifyStorageTransfers(program)) == 0 {
				t.Fatal("malformed private take passed verification")
			}
		})
	}
}

func TestPrivateVectorStorageTakeUnreachableAndProvenance(t *testing.T) {
	for _, missingProvenance := range []bool{false, true} {
		program := storageTakeMIR(t)
		for fi := range program.Functions {
			function := &program.Functions[fi]
			if function.Name != "main::take" {
				continue
			}
			if missingProvenance {
				function.UnsafeOperations = nil
				continue
			}
			var extra mir.Operation
			for _, block := range function.Blocks {
				for _, op := range block.Operations {
					if op.Kind == mir.OperationStorageTake {
						extra = op
					}
				}
			}
			extra.Type = "main::Missing"
			function.Blocks = append(function.Blocks, mir.Block{ID: mir.BlockID(len(function.Blocks)),
				Operations: []mir.Operation{extra}, Terminator: mir.Terminator{Kind: mir.TerminatorUnreachable}})
		}
		if len(mir.VerifyStorageTransfers(program)) == 0 {
			t.Fatalf("unreachable malformed take or missing provenance admitted: %v", missingProvenance)
		}
	}
}

func TestPrivateVectorStorageTakeBackendChecksPointerType(t *testing.T) {
	program := storageTakeMIR(t)
	for fi := range program.Functions {
		function := &program.Functions[fi]
		if function.Name != "main::take" {
			continue
		}
		for bi := range function.Blocks {
			for oi := range function.Blocks[bi].Operations {
				op := &function.Blocks[bi].Operations[oi]
				if op.Kind == mir.OperationStorageTake {
					op.Operands[0].Local = function.Parameters[1]
				}
			}
		}
	}
	if _, err := cbackend.EmitStorageTransfers(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil ||
		!strings.Contains(err.Error(), "private storage take requires an exact mutable void pointer") {
		t.Fatalf("backend accepted scalar source or failed outside pointer guard: %v", err)
	}
}

func TestPrivateVectorStorageTakeMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("storage-take execution requires Windows amd64")
	}
	unit, err := cbackend.EmitStorageTransfers(storageTakeMIR(t), "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil || !slices.Contains(unit.Features, "private-storage-take") {
		t.Fatalf("private emit: %v", err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalar+storageTakeProbe)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("storage-take execution: %+v %v", outcome, err)
			}
		})
	}
}

const storageTakeProbe = `
#include <stdlib.h>
#include <string.h>
static void *take_slot;
static void *take_live;
static unsigned take_exports, take_drops, take_bad;
void *fairyc_fixture_take_slot(void) {
    if (take_exports != 0) { take_bad++; return NULL; }
    int32_t *value = malloc(sizeof *value);
    if (value == NULL) { take_bad++; return NULL; }
    *value = 41;
    take_slot = take_live = value;
    take_exports++;
    return &take_slot;
}
int32_t fairyc_fixture_take_id(void *handle) {
    if (handle == NULL || handle != take_live) { take_bad++; return 0; }
    return *(int32_t *)handle;
}
uint32_t fairyc_fixture_take_drop(void *handle) {
    if (handle == NULL || handle != take_live || handle != take_slot) { take_bad++; return 4; }
    free(handle);
    take_live = NULL;
    take_drops++;
    return 0;
}
int32_t fairyc_fixture_take_finished(void) {
    return take_exports == 1 && take_drops == 1 && take_live == NULL && take_bad == 0;
}
`
