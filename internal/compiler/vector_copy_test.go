package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestPrivateVectorCopyScalars(t *testing.T) {
	for _, item := range []struct{ scalar, nativeType, value, other string }{
		{"bool", "int32_t", "true", "false"}, {"u8", "uint8_t", "255", "0"},
		{"i32", "int32_t", "-2147483647", "42"}, {"u32", "uint32_t", "4294967295", "0"}, {"usize", "uintptr_t", "65537", "0"},
	} {
		t.Run(item.scalar, func(t *testing.T) {
			source := strings.NewReplacer("SCALAR", item.scalar, "VALUE", item.value, "OTHER", item.other).Replace(vectorCopyProgram)
			runPrivateVectorStorageNativeWithLayout(t, source, vectorCopyProbe, hir.Type(item.scalar), item.nativeType)
		})
	}
}

func TestPrivateVectorCopyRejectsWrongScalar(t *testing.T) {
	text := strings.NewReplacer("SCALAR", "i32", "VALUE", "42", "OTHER", "0", "element_size()", "4", "element_alignment()", "4").Replace(vectorCopyProgram)
	text = strings.Replace(text, "push<i32>(ref mut storage, item)", "push<i32>(ref mut storage, false)", 1)
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err == nil && result.Verified {
		t.Fatal("wrong scalar accepted by typed push")
	}
}

const vectorCopyProgram = `module main;
@link_name("fairyc_fixture_vector_initialize") unsafe extern C fn u32 initialize(ptr mut void storage, usize size, usize alignment);
@link_name("fairyc_fixture_vector_drop") unsafe extern C fn u32 release(ptr mut void storage);
@link_name("fairyc_fixture_vector_finished") unsafe extern C fn i32 finished();
@link_name("fairyc_fixture_vector_push") unsafe extern C fn u32 push_raw(ptr mut void storage, ptr void value, usize size);
@link_name("fairyc_fixture_vector_read") unsafe extern C fn u32 read_raw(ptr void storage, usize index, ptr mut void output, usize size);
@link_name("fairyc_fixture_vector_pop") unsafe extern C fn u32 pop_raw(ptr mut void storage, ptr mut void output, usize size);
@link_name("fairyc_fixture_vector_fail_next") unsafe extern C fn i32 fail_next();
@link_name("fairyc_fixture_vector_wrong_size") unsafe extern C fn i32 wrong_size(ptr mut void storage);
resource Storage<T> {
    ptr mut void data; usize bytes; usize alignment; usize length; usize element_size; usize element_alignment;
    cleanup { let u32 status = unsafe { release(ref mut this as ptr mut void) }; }
}
fn Storage<T> empty<T>() {
    return Storage<T> { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 0, .element_alignment = 0 };
}
fn u32 push<T>(ref mut Storage<T> storage, T value) {
    return unsafe { push_raw(storage as ptr mut void, ref value as ptr void, storage.element_size) };
}
fn u32 read<T>(ref Storage<T> storage, usize index, ref mut T output) {
    return unsafe { read_raw(storage as ptr void, index, output as ptr mut void, storage.element_size) };
}
fn u32 pop<T>(ref mut Storage<T> storage, ref mut T output) {
    return unsafe { pop_raw(storage as ptr mut void, output as ptr mut void, storage.element_size) };
}
fn i32 exercise() {
    var Storage<SCALAR> storage = empty<SCALAR>();
    assert(unsafe { initialize(ref mut storage as ptr mut void, element_size(), element_alignment()) } == 0);
    let SCALAR item = VALUE;
    let SCALAR other = OTHER;
    var SCALAR output = other;
    assert(unsafe { fail_next() } == 1);
    assert(push<SCALAR>(ref mut storage, item) == 1 && storage.length == 0);
    assert(push<SCALAR>(ref mut storage, item) == 0 && storage.length == 1);
    assert(unsafe { wrong_size(ref mut storage as ptr mut void) } == 1);
    assert(read<SCALAR>(ref storage, 1, ref mut output) == 4 && output == other);
    for (var usize i = 1; i < 8; i += 1) { assert(push<SCALAR>(ref mut storage, item) == 0); }
    assert(unsafe { fail_next() } == 1);
    assert(push<SCALAR>(ref mut storage, other) == 1 && storage.length == 8);
    assert(read<SCALAR>(ref storage, 0, ref mut output) == 0 && output == item);
    assert(push<SCALAR>(ref mut storage, other) == 0 && storage.length == 9);
    assert(read<SCALAR>(ref storage, 8, ref mut output) == 0 && output == other);
    assert(pop<SCALAR>(ref mut storage, ref mut output) == 0 && output == other && storage.length == 8);
    output = item;
    assert(read<SCALAR>(ref storage, 8, ref mut output) == 4 && output == item);
    for (var usize i = 0; i < 8; i += 1) { assert(pop<SCALAR>(ref mut storage, ref mut output) == 0 && output == item); }
    assert(pop<SCALAR>(ref mut storage, ref mut output) == 4 && output == item && storage.length == 0);
    let Storage<SCALAR> moved = storage;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

const vectorCopyProbe = `
uint32_t fairyc_fixture_vector_push(void *storage, const void *value, size_t size) {
    return fy_private_vector_push_copy_v0(storage, value, size);
}
uint32_t fairyc_fixture_vector_read(const void *storage, size_t index, void *output, size_t size) {
    return fy_private_vector_read_copy_v0(storage, index, output, size);
}
uint32_t fairyc_fixture_vector_pop(void *storage, void *output, size_t size) {
    return fy_private_vector_pop_copy_v0(storage, output, size);
}
int32_t fairyc_fixture_vector_fail_next(void) { vector_fail = 1; return 1; }
int32_t fairyc_fixture_vector_wrong_size(void *storage) {
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    unsigned char before[sizeof value];
    memcpy(before, storage, sizeof before);
    uint64_t output = UINT64_C(123456789);
    size_t wrong = value.element_size == sizeof output ? 1 : value.element_size + 1;
    if (fy_private_vector_push_copy_v0(storage, &output, wrong) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        fy_private_vector_read_copy_v0(storage, 0, &output, wrong) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        fy_private_vector_pop_copy_v0(storage, &output, wrong) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        output != UINT64_C(123456789) || memcmp(before, storage, sizeof before) != 0 || vector_live != 1) return 0;
    return 1;
}
`
