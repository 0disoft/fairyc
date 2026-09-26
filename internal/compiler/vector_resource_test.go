package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestPrivateVectorResourceOwnership(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", vectorResourceProgram)
	before, _ := json.Marshal(file.Module)
	result, plan, err := compiler.CheckStorageTransferFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified || len(plan.Specializations) != 6 {
		t.Fatalf("source transfer: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("source transfer modified original AST")
	}
	ordinary, _, err := compiler.CheckStorageGenericFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err == nil && ordinary.Verified {
		t.Fatal("query-only source lane admitted storage take")
	}
	runVectorStorageMIRNativeLane(t, result.MIR, "main::main", vectorResourceProbe, true)
}

func TestPrivateVectorResourceDonorCannotBeReused(t *testing.T) {
	text := strings.Replace(vectorResourceProgram,
		"assert(recover_and_retry(ref mut storage, make(1), 1) == 1);",
		"let Item first = make(1); assert(recover_and_retry(ref mut storage, first, 1) == 1); assert(item_id(ref first) == 1);", 1)
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, _, err := compiler.CheckStorageTransferFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err == nil && result.Verified {
		t.Fatal("transferred resource donor remained usable")
	}
}

func TestPrivateVectorOptionDiscardOwnership(t *testing.T) {
	start := strings.Index(vectorResourceProgram, "    let Item last = switch")
	end := strings.Index(vectorResourceProgram, "    let Items<Item> moved")
	if start < 0 || end <= start {
		t.Fatal("option consumer fixture boundary missing")
	}
	text := vectorResourceProgram[:start] +
		"    assert({ let Option<Item> ignored = pop<Item>(ref mut storage); true });\n" +
		vectorResourceProgram[end:]
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, _, err := compiler.CheckStorageTransferFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("discarded option source: %v %+v", err, result.Semantic)
	}
	runVectorStorageMIRNativeLane(t, result.MIR, "main::main", vectorResourceProbe, true)
}

func TestPrivateVectorStorageTakeSourceRejections(t *testing.T) {
	for name, replacement := range map[string][2]string{
		"unsafe missing":     {"return unsafe { storage_take<T>(slot) };", "return storage_take<T>(slot);"},
		"type arity":         {"storage_take<T>(slot)", "storage_take<T, T>(slot)"},
		"missing pointer":    {"storage_take<T>(slot)", "storage_take<T>()"},
		"extra pointer":      {"storage_take<T>(slot)", "storage_take<T>(slot, slot)"},
		"scalar pointer":     {"storage_take<T>(slot)", "storage_take<T>(1)"},
		"readonly pointer":   {"take_slot<T>(ptr mut void slot)", "take_slot<T>(ptr void slot)"},
		"unknown type":       {"storage_take<T>(slot)", "storage_take<Missing>(slot)"},
		"unsupported type":   {"storage_take<T>(slot)", "storage_take<i64>(slot)"},
		"reserved function":  {"fn T take_slot<T>", "fn T storage_take<T>"},
		"reserved binding":   {"return unsafe { storage_take<T>(slot) };", "let i32 storage_take = 1; return unsafe { storage_take<T>(slot) };"},
		"popped donor reuse": {"let i32 discarded = discard_item(last);", "let i32 discarded = discard_item(last); assert(item_id(ref last) == 9);"},
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(vectorResourceProgram, replacement[0], replacement[1], 1)
			if text == vectorResourceProgram {
				t.Fatal("rejection fixture did not change")
			}
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			result, _, err := compiler.CheckStorageTransferFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("invalid storage take source admitted")
			}
			if name == "unsafe missing" {
				found := false
				for _, diagnostic := range result.Semantic.Diagnostics {
					found = found || diagnostic.Code == "E5101"
				}
				if !found {
					t.Fatalf("unsafe source failed without E5101: %v %+v", err, result.Semantic)
				}
			}
		})
	}
}

const vectorResourceProgram = `module main;
@link_name("fairyc_fixture_vector_initialize") unsafe extern C fn u32 initialize(ptr mut void storage, usize size, usize alignment);
@link_name("fairyc_fixture_vector_drop") unsafe extern C fn u32 release(ptr mut void storage);
@link_name("fairyc_fixture_resource_new") unsafe extern C fn ptr mut void new_raw(i32 id);
@link_name("fairyc_fixture_resource_drop") unsafe extern C fn u32 drop_raw(ptr mut void handle);
@link_name("fairyc_fixture_resource_id") unsafe extern C fn i32 id_raw(ptr mut void handle);
@link_name("fairyc_fixture_resource_prepare") unsafe extern C fn u32 prepare_raw(ptr mut void storage, usize size, ptr mut void output);
@link_name("fairyc_fixture_resource_commit") unsafe extern C fn u32 commit_raw(ptr mut void storage, ptr mut void slot, usize size);
@link_name("fairyc_fixture_resource_pop_slot") unsafe extern C fn ptr mut void pop_slot_raw(ptr mut void storage, usize size);
@link_name("fairyc_fixture_resource_fail") unsafe extern C fn i32 fail_next();
@link_name("fairyc_fixture_resource_finished") unsafe extern C fn i32 finished();
@link_name("fairyc_fixture_resource_empty") unsafe extern C fn i32 empty_valid(ptr void storage);
resource Item {
    ptr mut void handle;
    cleanup { let u32 status = unsafe { drop_raw(this.handle) }; }
}
resource Items<T> {
    ptr mut void data; usize bytes; usize alignment; usize length; usize element_size; usize element_alignment;
    cleanup {
        while (this.length > 0) {
            let ptr mut void slot = unsafe { pop_slot_raw(ref mut this as ptr mut void, this.element_size) };
            assert(slot != null);
            let T value = take_slot<T>(slot);
        }
        let u32 released = unsafe { release(ref mut this as ptr mut void) };
    }
}
fn T take_slot<T>(ptr mut void slot) { return unsafe { storage_take<T>(slot) }; }
fn i32 write_slot<T>(ptr mut void slot, T value) { return unsafe { storage_put<T>(slot, value); 1 }; }
fn Item make(i32 id) {
    let ptr mut void handle = unsafe { new_raw(id) };
    assert(handle != null);
    return Item { .handle = handle };
}
fn i32 item_id(ref Item value) { return unsafe { id_raw(value.handle) }; }
fn i32 discard_item(Item value) { return 0; }
fn Items<T> empty_items<T>() {
    var Items<T> storage = Items<T> { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 0, .element_alignment = 0 };
    assert(unsafe { initialize(ref mut storage as ptr mut void, storage_size<T>(), storage_alignment<T>()) } == 0);
    return storage;
}
fn Result<u32, T> push<T>(ref mut Items<T> storage, T value) {
    var ptr mut void slot = null;
    let u32 status = unsafe { prepare_raw(storage as ptr mut void, storage.element_size, ref mut slot as ptr mut void) };
    return if (status == 0) {
        assert(slot != null);
        assert(write_slot<T>(slot, value) == 1);
        assert(unsafe { commit_raw(storage as ptr mut void, slot, storage.element_size) } == 0);
        .ok(status)
    } else { .err(value) };
}
fn i32 push_ok(ref mut Items<Item> storage, Item value) {
    return switch (push<Item>(storage, value)) {
        case .ok(_) : { 1 }
        case .err(_) : { 0 }
    };
}
fn i32 recover_and_retry(ref mut Items<Item> storage, Item value, i32 expected) {
    return switch (push<Item>(storage, value)) {
        case .ok(_) : { 0 }
        case .err(returned) : {
            assert(item_id(ref returned) == expected);
            push_ok(storage, returned)
        }
    };
}
fn Option<T> pop<T>(ref mut Items<T> storage) {
    return if (storage.length == 0) { .none } else {
        let ptr mut void slot = unsafe { pop_slot_raw(storage as ptr mut void, storage.element_size) };
        assert(slot != null);
        .some(take_slot<T>(slot))
    };
}
fn i32 exercise() {
    var Items<Item> storage = empty_items<Item>();
    assert(unsafe { empty_valid(ref storage as ptr void) } == 1);
    assert(switch (pop<Item>(ref mut storage)) { case .some(_) : { false } case .none : { true } });
    assert(unsafe { pop_slot_raw(ref mut storage as ptr mut void, storage.element_size) } == null);
    assert(unsafe { fail_next() } == 1);
    assert(recover_and_retry(ref mut storage, make(1), 1) == 1);
    for (var i32 id = 2; id < 9; id += 1) { assert(push_ok(ref mut storage, make(id)) == 1); }
    assert(storage.length == 8);
    assert(unsafe { fail_next() } == 1);
    assert(recover_and_retry(ref mut storage, make(9), 9) == 1);
    assert(storage.length == 9);
    assert(unsafe { pop_slot_raw(ref mut storage as ptr mut void, storage.element_size + 1) } == null && storage.length == 9);
    let Item last = switch (pop<Item>(ref mut storage)) {
        case .some(value) : { value }
        case .none : { assert(false); }
    };
    assert(item_id(ref last) == 9);
    let i32 discarded = discard_item(last);
    let Items<Item> moved = storage;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

const vectorResourceProbe = `
#include <stdio.h>
int32_t fairyc_fixture_resource_empty(const void *storage) {
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    return vector_live == 0 && value.block.data == NULL && value.block.size == 0 &&
        value.block.alignment == 0 && value.length == 0 &&
        value.element_size == sizeof(void *) && value.element_alignment == _Alignof(void *);
}
_Static_assert(sizeof(void *) == 8 && _Alignof(void *) == 8, "fixture Item layout");
static void *resource_handles[9] = {NULL};
static size_t resource_live = 0;
static size_t resource_drops = 0;
static int resource_bad = 0;
static size_t resource_slots = 0;
static size_t resource_prepares = 0, resource_prepare_failures = 0, resource_commits = 0;
void *fairyc_fixture_resource_new(int32_t id) {
    if (id < 1 || id > 9 || resource_handles[id - 1] != NULL) { resource_bad = 1; return NULL; }
    int32_t *handle = malloc(sizeof *handle);
    if (handle != NULL) { *handle = id; resource_handles[id - 1] = handle; resource_live++; }
    return handle;
}
uint32_t fairyc_fixture_resource_drop(void *handle) {
    if (handle == NULL) { resource_bad = 1; return 4; }
    for (size_t i = 0; i < 9; i++) {
        if (resource_handles[i] == handle) {
            if (resource_drops >= 9 || i + 1 != 9 - resource_drops) resource_bad = 1;
            resource_handles[i] = NULL;
            resource_drops++;
            resource_live--;
            free(handle);
            return 0;
        }
    }
    resource_bad = 1;
    return 4;
}
int32_t fairyc_fixture_resource_id(void *handle) {
    for (size_t i = 0; i < 9; i++) {
        if (handle != NULL && resource_handles[i] == handle) return (int32_t)(i + 1);
    }
    resource_bad = 1;
    return 0;
}
uint32_t fairyc_fixture_resource_prepare(void *storage, size_t size, void *output) {
    struct fy_private_vector_storage_v0 before, after;
    void *prior, *slot;
    size_t live = vector_live;
    memcpy(&before, storage, sizeof before);
    memcpy(&prior, output, sizeof prior);
    if (fy_private_vector_prepare_push_v0(NULL, size, output) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        fy_private_vector_prepare_push_v0(storage, size, NULL) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        fy_private_vector_prepare_push_v0(storage, size + 1, output) != FY_STD_ALLOC_INVALID_ARGUMENT_V0) resource_bad = 1;
    memcpy(&after, storage, sizeof after);
    memcpy(&slot, output, sizeof slot);
    if (memcmp(&before, &after, sizeof before) != 0 || slot != prior || vector_live != live) resource_bad = 1;
    uint32_t status = fy_private_vector_prepare_push_v0(storage, size, output);
    memcpy(&after, storage, sizeof after);
    memcpy(&slot, output, sizeof slot);
    if (status != 0) {
        if (memcmp(&before, &after, sizeof before) != 0 || slot != prior || vector_live != live) resource_bad = 1;
        resource_prepare_failures++;
    } else {
        if (after.length != before.length || after.element_size != size || after.element_alignment != before.element_alignment ||
            slot != (unsigned char *)after.block.data + after.length * size) resource_bad = 1;
        resource_prepares++;
    }
    return status;
}
uint32_t fairyc_fixture_resource_commit(void *storage, void *slot, size_t size) {
    struct fy_private_vector_storage_v0 before, after;
    memcpy(&before, storage, sizeof before);
    if (fy_private_vector_commit_push_v0(storage, NULL, size) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        fy_private_vector_commit_push_v0(storage, slot, size + 1) != FY_STD_ALLOC_INVALID_ARGUMENT_V0) resource_bad = 1;
    memcpy(&after, storage, sizeof after);
    if (memcmp(&before, &after, sizeof before) != 0) resource_bad = 1;
    void *handle;
    memcpy(&handle, slot, sizeof handle);
    if (fairyc_fixture_resource_id(handle) != (int32_t)(before.length + 1) || resource_drops != 0) resource_bad = 1;
    uint32_t status = fy_private_vector_commit_push_v0(storage, slot, size);
    memcpy(&after, storage, sizeof after);
    before.length++;
    if (status != 0 || memcmp(&before, &after, sizeof before) != 0) resource_bad = 1;
    if (fy_private_vector_commit_push_v0(storage, slot, size) != FY_STD_ALLOC_INVALID_ARGUMENT_V0) resource_bad = 1;
    memcpy(&after, storage, sizeof after);
    if (memcmp(&before, &after, sizeof before) != 0) resource_bad = 1;
    resource_commits++;
    return status;
}
void *fairyc_fixture_resource_pop_slot(void *storage, size_t size) {
    struct fy_private_vector_storage_v0 before, after;
    memcpy(&before, storage, sizeof before);
    void *slot = fy_private_vector_pop_slot_v0(storage, size);
    memcpy(&after, storage, sizeof after);
    if (slot != NULL) {
        if (before.length == 0 || slot != (unsigned char *)before.block.data + (before.length - 1) * size) resource_bad = 1;
        before.length--;
        resource_slots++;
    }
    if (memcmp(&before, &after, sizeof before) != 0) resource_bad = 1;
    return slot;
}
int32_t fairyc_fixture_resource_fail(void) { vector_fail = 1; return 1; }
int32_t fairyc_fixture_resource_finished(void) {
    if (resource_live != 0 || resource_drops != 9 || resource_bad) {
        fprintf(stderr, "resource live=%zu drops=%zu bad=%d slots=%zu prepares=%zu failures=%zu commits=%zu\n",
            resource_live, resource_drops, resource_bad, resource_slots, resource_prepares, resource_prepare_failures, resource_commits);
    }
    return fairyc_fixture_vector_finished() == 1 && resource_live == 0 && resource_drops == 9 &&
        resource_slots == 9 && resource_prepares == 9 && resource_prepare_failures == 2 && resource_commits == 9 && !resource_bad ? 1 : 0;
}
`
