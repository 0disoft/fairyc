package cbackendnative

const ownedTextSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_system_heap_block_v0 {
    ptr mut void data;
    c::size_t size;
    c::size_t alignment;
}
@repr(C)
struct fy_std_buffer_v0 {
    fy_std_system_heap_block_v0 block;
    c::size_t length;
}
@repr(C)
struct fy_std_text_view_v0 {
    ptr u8 data;
    c::size_t length;
}
@repr(C)
struct fy_std_string_v0 {
    fy_std_buffer_v0 bytes;
}
@repr(C)
struct fy_std_cstr_view_v0 {
    ptr u8 data;
    c::size_t length;
}
@repr(C)
struct fy_std_cstring_v0 {
    fy_std_buffer_v0 bytes;
}
@repr(C)
struct fy_std_owned_text_result_v0 {
    u32 status;
    u32 allocation_status;
}

@link_name("fairyc_fixture_owned_text_zero_size_t")
unsafe extern C fn c::size_t zero_size_t() ;
@link_name("fairyc_fixture_owned_text_storage")
unsafe extern C fn ptr u8 text_storage() ;
@link_name("fairyc_fixture_owned_text_storage_length")
unsafe extern C fn c::size_t text_storage_length() ;

@link_name("fy_std_string_copy_v0")
unsafe extern C fn fy_std_owned_text_result_v0 string_copy(ptr fy_std_text_view_v0 input, ptr mut fy_std_string_v0 output) ;
@link_name("fy_std_string_as_view_v0")
unsafe extern C fn fy_std_owned_text_result_v0 string_as_view(ptr fy_std_string_v0 input, ptr mut fy_std_text_view_v0 output) ;
@link_name("fy_std_string_append_v0")
unsafe extern C fn fy_std_owned_text_result_v0 string_append(ptr mut fy_std_string_v0 value, ptr fy_std_text_view_v0 suffix) ;
@link_name("fy_std_string_drop_v0")
unsafe extern C fn fy_std_owned_text_result_v0 string_drop(ptr mut fy_std_string_v0 value) ;

@link_name("fy_std_cstring_from_text_v0")
unsafe extern C fn fy_std_owned_text_result_v0 cstring_from_text(ptr fy_std_text_view_v0 input, ptr mut fy_std_cstring_v0 output) ;
@link_name("fy_std_cstring_as_view_v0")
unsafe extern C fn fy_std_owned_text_result_v0 cstring_as_view(ptr fy_std_cstring_v0 input, ptr mut fy_std_cstr_view_v0 output) ;
@link_name("fy_std_cstring_drop_v0")
unsafe extern C fn fy_std_owned_text_result_v0 cstring_drop(ptr mut fy_std_cstring_v0 value) ;

@link_name("fairyc_fixture_string_view_is_fairy")
unsafe extern C fn u32 string_view_is_fairy(ptr fy_std_text_view_v0 view) ;
@link_name("fairyc_fixture_string_view_is_fairy_twice")
unsafe extern C fn u32 string_view_is_fairy_twice(ptr fy_std_text_view_v0 view) ;
@link_name("fairyc_fixture_string_is_empty")
unsafe extern C fn u32 string_is_empty(ptr fy_std_string_v0 value) ;
@link_name("fairyc_fixture_cstr_view_is_fairy")
unsafe extern C fn u32 cstr_view_is_fairy(ptr fy_std_cstr_view_v0 view) ;
@link_name("fairyc_fixture_cstring_is_empty")
unsafe extern C fn u32 cstring_is_empty(ptr fy_std_cstring_v0 value) ;

fn i32 main() {
    let c::size_t zero = unsafe { zero_size_t() };
    let ptr u8 storage = unsafe { text_storage() };
    let c::size_t storage_length = unsafe { text_storage_length() };
    let fy_std_text_view_v0 input = fy_std_text_view_v0 {
        .data = storage,
        .length = storage_length,
    };

    let fy_std_system_heap_block_v0 string_block = fy_std_system_heap_block_v0 {
        .data = null,
        .size = zero,
        .alignment = zero,
    };
    let fy_std_buffer_v0 string_buffer = fy_std_buffer_v0 {
        .block = string_block,
        .length = zero,
    };
    var fy_std_string_v0 owned_string = fy_std_string_v0 { .bytes = string_buffer };
    var fy_std_text_view_v0 string_view = fy_std_text_view_v0 {
        .data = null,
        .length = zero,
    };
    let fy_std_owned_text_result_v0 copied = unsafe {
        string_copy(
            ref input as ptr fy_std_text_view_v0,
            ref mut owned_string as ptr mut fy_std_string_v0
        )
    };
    let fy_std_owned_text_result_v0 viewed_once = unsafe {
        string_as_view(
            ref owned_string as ptr fy_std_string_v0,
            ref mut string_view as ptr mut fy_std_text_view_v0
        )
    };
    let u32 observed_once = unsafe {
        string_view_is_fairy(ref string_view as ptr fy_std_text_view_v0)
    };
    let fy_std_owned_text_result_v0 appended = unsafe {
        string_append(
            ref mut owned_string as ptr mut fy_std_string_v0,
            ref input as ptr fy_std_text_view_v0
        )
    };
    let fy_std_owned_text_result_v0 viewed_twice = unsafe {
        string_as_view(
            ref owned_string as ptr fy_std_string_v0,
            ref mut string_view as ptr mut fy_std_text_view_v0
        )
    };
    let u32 observed_twice = unsafe {
        string_view_is_fairy_twice(ref string_view as ptr fy_std_text_view_v0)
    };
    let fy_std_owned_text_result_v0 dropped_string = unsafe {
        string_drop(ref mut owned_string as ptr mut fy_std_string_v0)
    };
    let u32 string_empty = unsafe {
        string_is_empty(ref owned_string as ptr fy_std_string_v0)
    };

    let fy_std_system_heap_block_v0 cstring_block = fy_std_system_heap_block_v0 {
        .data = null,
        .size = zero,
        .alignment = zero,
    };
    let fy_std_buffer_v0 cstring_buffer = fy_std_buffer_v0 {
        .block = cstring_block,
        .length = zero,
    };
    var fy_std_cstring_v0 owned_cstring = fy_std_cstring_v0 { .bytes = cstring_buffer };
    var fy_std_cstr_view_v0 cstr_view = fy_std_cstr_view_v0 {
        .data = null,
        .length = zero,
    };
    let fy_std_owned_text_result_v0 created_cstring = unsafe {
        cstring_from_text(
            ref input as ptr fy_std_text_view_v0,
            ref mut owned_cstring as ptr mut fy_std_cstring_v0
        )
    };
    let fy_std_owned_text_result_v0 viewed_cstring = unsafe {
        cstring_as_view(
            ref owned_cstring as ptr fy_std_cstring_v0,
            ref mut cstr_view as ptr mut fy_std_cstr_view_v0
        )
    };
    let u32 observed_cstring = unsafe {
        cstr_view_is_fairy(ref cstr_view as ptr fy_std_cstr_view_v0)
    };
    let fy_std_owned_text_result_v0 dropped_cstring = unsafe {
        cstring_drop(ref mut owned_cstring as ptr mut fy_std_cstring_v0)
    };
    let u32 cstring_empty = unsafe {
        cstring_is_empty(ref owned_cstring as ptr fy_std_cstring_v0)
    };

    return if (copied.status == 0 && copied.allocation_status == 0 &&
        viewed_once.status == 0 && viewed_once.allocation_status == 0 && observed_once == 1 &&
        appended.status == 0 && appended.allocation_status == 0 &&
        viewed_twice.status == 0 && viewed_twice.allocation_status == 0 && observed_twice == 1 &&
        dropped_string.status == 0 && dropped_string.allocation_status == 0 && string_empty == 1 &&
        created_cstring.status == 0 && created_cstring.allocation_status == 0 &&
        viewed_cstring.status == 0 && viewed_cstring.allocation_status == 0 && observed_cstring == 1 &&
        dropped_cstring.status == 0 && dropped_cstring.allocation_status == 0 && cstring_empty == 1) {
        42
    } else {
        0
    };
}
`

const ownedTextSourceCallBridge = `
static const uint8_t fairyc_fixture_owned_text_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)
};

size_t fairyc_fixture_owned_text_zero_size_t(void) { return (size_t)0; }

const uint8_t *fairyc_fixture_owned_text_storage(void) {
    return fairyc_fixture_owned_text_bytes;
}

size_t fairyc_fixture_owned_text_storage_length(void) {
    return sizeof(fairyc_fixture_owned_text_bytes);
}

static uint32_t fairyc_fixture_bytes_are_fairy(const uint8_t *data, size_t length) {
    return data != NULL && length == (size_t)5 &&
           data[0] == UINT8_C(70) && data[1] == UINT8_C(97) &&
           data[2] == UINT8_C(105) && data[3] == UINT8_C(114) &&
           data[4] == UINT8_C(121) ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_string_view_is_fairy(const struct fy_std_text_view_v0 *view) {
    return view != NULL ? fairyc_fixture_bytes_are_fairy(view->data, view->length) : UINT32_C(0);
}

uint32_t fairyc_fixture_string_view_is_fairy_twice(const struct fy_std_text_view_v0 *view) {
    return view != NULL && view->data != NULL && view->length == (size_t)10 &&
           fairyc_fixture_bytes_are_fairy(view->data, (size_t)5) == UINT32_C(1) &&
           fairyc_fixture_bytes_are_fairy(view->data + (size_t)5, (size_t)5) == UINT32_C(1) ?
           UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_string_is_empty(const struct fy_std_string_v0 *value) {
    return value != NULL && value->bytes.block.data == NULL &&
           value->bytes.block.size == (size_t)0 && value->bytes.block.alignment == (size_t)0 &&
           value->bytes.length == (size_t)0 ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_cstr_view_is_fairy(const struct fy_std_cstr_view_v0 *view) {
    return view != NULL && fairyc_fixture_bytes_are_fairy(view->data, view->length) == UINT32_C(1) &&
           view->data[view->length] == UINT8_C(0) ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_cstring_is_empty(const struct fy_std_cstring_v0 *value) {
    return value != NULL && value->bytes.block.data == NULL &&
           value->bytes.block.size == (size_t)0 && value->bytes.block.alignment == (size_t)0 &&
           value->bytes.length == (size_t)0 ? UINT32_C(1) : UINT32_C(0);
}
`

var ownedTextSourceCallExpectedCFragments = []string{
	"struct fy_std_owned_text_result_v0 {",
	"uint32_t status;",
	"uint32_t allocation_status;",
	"_Static_assert(sizeof(struct fy_std_owned_text_result_v0) == UINT64_C(8)",
	"struct fy_std_string_v0 {",
	"struct fy_std_buffer_v0 bytes;",
	"struct fy_std_cstring_v0 {",
	"extern struct fy_std_owned_text_result_v0 fy_std_string_copy_v0(const struct fy_std_text_view_v0 * fy_a_0, struct fy_std_string_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_string_as_view_v0(const struct fy_std_string_v0 * fy_a_0, struct fy_std_text_view_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_string_append_v0(struct fy_std_string_v0 * fy_a_0, const struct fy_std_text_view_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_string_drop_v0(struct fy_std_string_v0 * fy_a_0);",
	"extern struct fy_std_owned_text_result_v0 fy_std_cstring_from_text_v0(const struct fy_std_text_view_v0 * fy_a_0, struct fy_std_cstring_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_cstring_as_view_v0(const struct fy_std_cstring_v0 * fy_a_0, struct fy_std_cstr_view_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_cstring_drop_v0(struct fy_std_cstring_v0 * fy_a_0);",
	" = fy_std_string_copy_v0(",
	" = fy_std_cstring_from_text_v0(",
}
