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
	if err != nil || !result.Verified || len(plan.Specializations) != 1 {
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
@link_name("fairyc_fixture_resource_push") unsafe extern C fn u32 push_raw(ptr mut void storage, ptr mut void value, ptr void empty, usize size);
@link_name("fairyc_fixture_resource_pop_slot") unsafe extern C fn ptr mut void pop_slot_raw(ptr mut void storage, usize size);
@link_name("fairyc_fixture_resource_empty_item") unsafe extern C fn i32 empty_item_called();
@link_name("fairyc_fixture_resource_fail") unsafe extern C fn i32 fail_next();
@link_name("fairyc_fixture_resource_finished") unsafe extern C fn i32 finished();
@link_name("fairyc_fixture_resource_empty") unsafe extern C fn i32 empty_valid(ptr void storage);
resource Item {
    ptr mut void handle;
    cleanup { let u32 status = unsafe { drop_raw(this.handle) }; }
}
resource Items {
    ptr mut void data; usize bytes; usize alignment; usize length; usize element_size; usize element_alignment;
    cleanup {
        while (this.length > 0) {
            let ptr mut void slot = unsafe { pop_slot_raw(ref mut this as ptr mut void, this.element_size) };
            assert(slot != null);
            let Item value = take_slot<Item>(slot);
        }
        let u32 released = unsafe { release(ref mut this as ptr mut void) };
    }
}
fn Item empty_item() { assert(unsafe { empty_item_called() } == 1); return Item { .handle = null }; }
fn T take_slot<T>(ptr mut void slot) { return unsafe { storage_take<T>(slot) }; }
fn Item make(i32 id) {
    let ptr mut void handle = unsafe { new_raw(id) };
    assert(handle != null);
    return Item { .handle = handle };
}
fn i32 item_id(ref Item value) { return unsafe { id_raw(value.handle) }; }
fn i32 discard_item(Item value) { return 0; }
fn Items empty_items() {
    var Items storage = Items { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 0, .element_alignment = 0 };
    assert(unsafe { initialize(ref mut storage as ptr mut void, storage_size<Item>(), storage_alignment<Item>()) } == 0);
    return storage;
}
fn Result<u32, Item> push(ref mut Items storage, Item value) {
    var Item owned = value;
    let Item vacant = empty_item();
    let u32 status = unsafe { push_raw(storage as ptr mut void, ref mut owned as ptr mut void, ref vacant as ptr void, storage.element_size) };
    return if (status == 0) {
        let i32 discarded = discard_item(owned);
        .ok(status)
    } else { .err(owned) };
}
fn i32 push_ok(ref mut Items storage, Item value) {
    return switch (push(storage, value)) {
        case .ok(_) : { 1 }
        case .err(_) : { 0 }
    };
}
fn i32 recover_and_retry(ref mut Items storage, Item value, i32 expected) {
    return switch (push(storage, value)) {
        case .ok(_) : { 0 }
        case .err(returned) : {
            assert(item_id(ref returned) == expected);
            push_ok(storage, returned)
        }
    };
}
fn Item pop(ref mut Items storage) {
    let ptr mut void slot = unsafe { pop_slot_raw(storage as ptr mut void, storage.element_size) };
    assert(slot != null);
    return take_slot<Item>(slot);
}
fn i32 exercise() {
    var Items storage = empty_items();
    assert(unsafe { empty_valid(ref storage as ptr void) } == 1);
    assert(unsafe { pop_slot_raw(ref mut storage as ptr mut void, storage.element_size) } == null);
    assert(unsafe { fail_next() } == 1);
    assert(recover_and_retry(ref mut storage, make(1), 1) == 1);
    for (var i32 id = 2; id < 9; id += 1) { assert(push_ok(ref mut storage, make(id)) == 1); }
    assert(storage.length == 8);
    assert(unsafe { fail_next() } == 1);
    assert(recover_and_retry(ref mut storage, make(9), 9) == 1);
    assert(storage.length == 9);
    assert(unsafe { pop_slot_raw(ref mut storage as ptr mut void, storage.element_size + 1) } == null && storage.length == 9);
    let Item last = pop(ref mut storage);
    assert(item_id(ref last) == 9);
    let i32 discarded = discard_item(last);
    let Items moved = storage;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

const vectorResourceProbe = `
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
static size_t resource_empty_constructs = 0;
static size_t resource_slots = 0;
int32_t fairyc_fixture_resource_empty_item(void) { resource_empty_constructs++; return 1; }
void *fairyc_fixture_resource_new(int32_t id) {
    if (id < 1 || id > 9 || resource_handles[id - 1] != NULL) { resource_bad = 1; return NULL; }
    int32_t *handle = malloc(sizeof *handle);
    if (handle != NULL) { *handle = id; resource_handles[id - 1] = handle; resource_live++; }
    return handle;
}
uint32_t fairyc_fixture_resource_drop(void *handle) {
    if (handle == NULL) return 0;
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
uint32_t fairyc_fixture_resource_push(void *storage, void *value, const void *empty, size_t size) {
    return fy_private_vector_push_take_v0(storage, value, empty, size);
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
    return fairyc_fixture_vector_finished() == 1 && resource_live == 0 && resource_drops == 9 &&
        resource_slots == 9 && resource_empty_constructs == 11 && !resource_bad ? 1 : 0;
}
`
