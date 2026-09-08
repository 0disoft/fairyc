package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestPrivateVectorResourceOwnership(t *testing.T) {
	runPrivateVectorStorageNative(t, vectorResourceProgram, vectorResourceProbe)
}

func TestPrivateVectorResourceDonorCannotBeReused(t *testing.T) {
	text := strings.Replace(vectorResourceProgram,
		"assert(recover_and_retry(ref mut storage, make(1), 1) == 1);",
		"let Item first = make(1); assert(recover_and_retry(ref mut storage, first, 1) == 1); assert(item_id(ref first) == 1);", 1)
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err == nil && result.Verified {
		t.Fatal("transferred resource donor remained usable")
	}
}

const vectorResourceProgram = `module main;
@link_name("fairyc_fixture_vector_initialize") unsafe extern C fn u32 initialize(ptr mut void storage, usize size, usize alignment);
@link_name("fairyc_fixture_vector_drop") unsafe extern C fn u32 release(ptr mut void storage);
@link_name("fairyc_fixture_resource_new") unsafe extern C fn ptr mut void new_raw(i32 id);
@link_name("fairyc_fixture_resource_drop") unsafe extern C fn u32 drop_raw(ptr mut void handle);
@link_name("fairyc_fixture_resource_id") unsafe extern C fn i32 id_raw(ptr mut void handle);
@link_name("fairyc_fixture_resource_push") unsafe extern C fn u32 push_raw(ptr mut void storage, ptr mut void value, ptr void empty, usize size);
@link_name("fairyc_fixture_resource_pop") unsafe extern C fn u32 pop_raw(ptr mut void storage, ptr mut void output, usize size);
@link_name("fairyc_fixture_resource_fail") unsafe extern C fn i32 fail_next();
@link_name("fairyc_fixture_resource_finished") unsafe extern C fn i32 finished();
resource Item {
    ptr mut void handle;
    cleanup { let u32 status = unsafe { drop_raw(this.handle) }; }
}
resource Items {
    ptr mut void data; usize bytes; usize alignment; usize length; usize element_size; usize element_alignment;
    cleanup {
        while (this.length > 0) {
            var Item value = empty_item();
            let u32 status = unsafe { pop_raw(ref mut this as ptr mut void, ref mut value as ptr mut void, 8) };
            if (status != 0) { break; }
        }
        let u32 released = unsafe { release(ref mut this as ptr mut void) };
    }
}
fn Item empty_item() { return Item { .handle = null }; }
fn Item make(i32 id) {
    let ptr mut void handle = unsafe { new_raw(id) };
    assert(handle != null);
    return Item { .handle = handle };
}
fn i32 item_id(ref Item value) { return unsafe { id_raw(value.handle) }; }
fn i32 discard_item(Item value) { return 0; }
fn Items empty_items() {
    return Items { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 0, .element_alignment = 0 };
}
fn Result<u32, Item> push(ref mut Items storage, Item value) {
    var Item owned = value;
    let Item vacant = empty_item();
    let u32 status = unsafe { push_raw(storage as ptr mut void, ref mut owned as ptr mut void, ref vacant as ptr void, 8) };
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
    var Item output = empty_item();
    assert(unsafe { pop_raw(storage as ptr mut void, ref mut output as ptr mut void, 8) } == 0);
    return output;
}
fn i32 exercise() {
    var Items storage = empty_items();
    assert(unsafe { initialize(ref mut storage as ptr mut void, 8, 8) } == 0);
    assert(unsafe { fail_next() } == 1);
    assert(recover_and_retry(ref mut storage, make(1), 1) == 1);
    for (var i32 id = 2; id < 9; id += 1) { assert(push_ok(ref mut storage, make(id)) == 1); }
    assert(storage.length == 8);
    assert(unsafe { fail_next() } == 1);
    assert(recover_and_retry(ref mut storage, make(9), 9) == 1);
    assert(storage.length == 9);
    let Item last = pop(ref mut storage);
    assert(item_id(ref last) == 9);
    let i32 discarded = discard_item(last);
    let Items moved = storage;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

const vectorResourceProbe = `
_Static_assert(sizeof(void *) == 8 && _Alignof(void *) == 8, "fixture Item layout");
static void *resource_handles[9] = {NULL};
static size_t resource_live = 0;
static size_t resource_drops = 0;
static int resource_bad = 0;
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
uint32_t fairyc_fixture_resource_pop(void *storage, void *output, size_t size) {
    return fy_private_vector_pop_take_v0(storage, output, size);
}
int32_t fairyc_fixture_resource_fail(void) { vector_fail = 1; return 1; }
int32_t fairyc_fixture_resource_finished(void) {
    return fairyc_fixture_vector_finished() == 1 && resource_live == 0 && resource_drops == 9 && !resource_bad ? 1 : 0;
}
`
