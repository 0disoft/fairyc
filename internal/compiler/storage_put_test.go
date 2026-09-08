package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

const storagePutSource = `module main;
@link_name("fairyc_fixture_put_slot") unsafe extern C fn ptr mut void slot_raw();
@link_name("fairyc_fixture_put_new") unsafe extern C fn ptr mut void new_raw();
@link_name("fairyc_fixture_put_drop") unsafe extern C fn u32 drop_raw(ptr mut void handle);
@link_name("fairyc_fixture_put_consume") unsafe extern C fn i32 consume_raw();
@link_name("fairyc_fixture_put_finished") unsafe extern C fn i32 finished();
resource Item { ptr mut void handle; cleanup { let u32 status = unsafe { drop_raw(this.handle) }; } }
fn Item make() { let ptr mut void handle = unsafe { new_raw() }; assert(handle != null); return Item { .handle = handle }; }
fn i32 write<T>(ptr mut void slot, T item) { return unsafe { storage_put<T>(slot, item); 1 }; }
fn i32 exercise() {
    let ptr mut void slot = unsafe { slot_raw() };
    let Item value = make();
    assert(write<Item>(slot, value) == 1);
    return unsafe { consume_raw() };
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

func storagePutMIR(t *testing.T) mir.Program {
	t.Helper()
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", storagePutSource)
	before, _ := json.Marshal(file.Module)
	result, plan, err := compiler.CheckStorageTransferFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified || len(plan.Specializations) != 1 {
		t.Fatalf("source storage put: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("source storage put mutated the original AST")
	}
	return result.MIR
}

func TestPrivateVectorStoragePutSourceAndMSVC(t *testing.T) {
	program := storagePutMIR(t)
	if len(mir.Verify(program)) == 0 || len(mir.VerifyStorageQueries(program)) == 0 {
		t.Fatal("ordinary/query-only MIR admitted storage put")
	}
	if _, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
		t.Fatal("ordinary C emitter admitted storage put")
	}
	if _, err := interpreter.Execute(program, "main::main"); err == nil {
		t.Fatal("interpreter admitted storage put")
	}
	if _, err := mircodec.Encode(program, 1<<20); err == nil {
		t.Fatal("artifact encoder admitted storage put")
	}
	encoded, err := json.Marshal(map[string]any{"schema": mircodec.Schema, "mirVersion": mircodec.MIRVersion, "program": program})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mircodec.Decode(encoded, 1<<20); err == nil {
		t.Fatal("artifact decoder admitted storage put")
	}
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", storagePutSource)
	ordinary, _, err := compiler.CheckStorageGenericFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err == nil && ordinary.Verified {
		t.Fatal("query-only source admitted storage put")
	}
	runVectorStorageMIRNativeLane(t, program, "main::main", storagePutProbe, true)
}

func TestPrivateVectorStoragePutSourceRejections(t *testing.T) {
	for name, replacement := range map[string][2]string{
		"unsafe missing":     {"return unsafe { storage_put<T>(slot, item); 1 };", "storage_put<T>(slot, item); return 1;"},
		"value type":         {"storage_put<T>(slot, item)", "storage_put<T>(slot, false)"},
		"pointer type":       {"storage_put<T>(slot, item)", "storage_put<T>(1, item)"},
		"arity":              {"storage_put<T>(slot, item)", "storage_put<T>(slot)"},
		"type arity":         {"storage_put<T>(slot, item)", "storage_put<T, T>(slot, item)"},
		"reserved name":      {"fn i32 write<T>", "fn i32 storage_put<T>"},
		"callee donor reuse": {"storage_put<T>(slot, item); 1", "storage_put<T>(slot, item); let ref T again = ref item; 1"},
		"caller donor reuse": {"return unsafe { consume_raw() };", "let ref Item again = ref value; return unsafe { consume_raw() };"},
		"active borrow":      {"storage_put<T>(slot, item); 1", "let ref T loan = ref item; storage_put<T>(slot, item); let ref T again = loan; 1"},
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(storagePutSource, replacement[0], replacement[1], 1)
			if text == storagePutSource {
				t.Fatal("put rejection fixture did not change")
			}
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			result, _, err := compiler.CheckStorageTransferFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("invalid storage put source admitted")
			}
			code := ""
			if name == "unsafe missing" {
				code = "E5101"
			}
			if name == "active borrow" {
				code = "E2105"
			}
			if code != "" {
				found := false
				for _, diagnostic := range result.Semantic.Diagnostics {
					found = found || diagnostic.Code == code
				}
				if !found {
					t.Fatalf("missing %s: %v %+v", code, err, result.Semantic)
				}
			}
		})
	}
}

func TestPrivateVectorStoragePutBackendPointerGuard(t *testing.T) {
	program := storagePutMIR(t)
	for fi := range program.Functions {
		for bi := range program.Functions[fi].Blocks {
			for oi := range program.Functions[fi].Blocks[bi].Operations {
				op := &program.Functions[fi].Blocks[bi].Operations[oi]
				if op.Kind == mir.OperationStoragePut {
					op.Operands[0], op.Operands[1] = op.Operands[1], op.Operands[0]
				}
			}
		}
	}
	if _, err := cbackend.EmitStorageTransfers(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil ||
		!strings.Contains(err.Error(), "private storage put has incompatible pointer/value types") {
		t.Fatalf("wrong storage-put pointer reached emission or another guard: %v", err)
	}
}

func TestPrivateVectorStoragePutMIRGuards(t *testing.T) {
	for name, mutate := range map[string]func(*mir.Operation){
		"destination":  func(op *mir.Operation) { op.DestinationLocal = 99 },
		"unknown type": func(op *mir.Operation) { op.OwnerType = "main::Missing" },
		"scope":        func(op *mir.Operation) { op.UnsafeScope = hir.Span{} },
		"operand":      func(op *mir.Operation) { op.Operands = nil },
		"value result": func(op *mir.Operation) { op.Type = hir.TypeI32 },
		"field":        func(op *mir.Operation) { op.Field = "handle" },
	} {
		t.Run(name, func(t *testing.T) {
			program := storagePutMIR(t)
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					for oi := range program.Functions[fi].Blocks[bi].Operations {
						op := &program.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind == mir.OperationStoragePut {
							mutate(op)
						}
					}
				}
			}
			if len(mir.VerifyStorageTransfers(program)) == 0 {
				t.Fatal("malformed storage put admitted")
			}
		})
	}
	program := storagePutMIR(t)
	for fi := range program.Functions {
		for bi := range program.Functions[fi].Blocks {
			block := &program.Functions[fi].Blocks[bi]
			for oi, op := range block.Operations {
				if op.Kind == mir.OperationStoragePut {
					reuse := mir.Operation{Kind: mir.OperationAssign, DestinationLocal: 999,
						Operands: []mir.Operand{op.Operands[1]}, Type: op.OwnerType}
					block.Operations = append(block.Operations[:oi+1], append([]mir.Operation{reuse}, block.Operations[oi+1:]...)...)
					break
				}
			}
		}
	}
	if len(mir.VerifyStorageTransfers(program)) == 0 {
		t.Fatal("put's consumed MIR temporary remained usable")
	}
}

const storagePutProbe = `
#include <stdlib.h>
static void *put_slot;
static void *put_live;
static unsigned put_requests, put_drops, put_consumes, put_bad;
void *fairyc_fixture_put_slot(void) {
    if (put_requests++ != 0) put_bad++;
    return &put_slot;
}
void *fairyc_fixture_put_new(void) {
    if (put_live != NULL) { put_bad++; return NULL; }
    int32_t *value = malloc(sizeof *value);
    if (value != NULL) { *value = 41; put_live = value; }
    return value;
}
uint32_t fairyc_fixture_put_drop(void *handle) {
    if (handle == NULL || handle != put_live) { put_bad++; return 4; }
    free(handle); put_live = NULL; put_drops++; return 0;
}
int32_t fairyc_fixture_put_consume(void) {
    if (put_slot == NULL || put_slot != put_live || put_drops != 0 || put_consumes != 0) { put_bad++; return 0; }
    int32_t result = *(int32_t *)put_slot;
    if (fairyc_fixture_put_drop(put_slot) != 0) return 0;
    put_slot = NULL; put_consumes++; return result;
}
int32_t fairyc_fixture_put_finished(void) {
    return put_requests == 1 && put_drops == 1 && put_consumes == 1 && put_live == NULL && put_bad == 0;
}
`
