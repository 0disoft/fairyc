package cbackendnative

const bufferSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_buffer_v0 {
    fy_std_system_heap_block_v0 block;
    c::size_t length;
}
@repr(C)
struct fy_std_system_heap_block_v0 {
    ptr mut void data;
    c::size_t size;
    c::size_t alignment;
}

@link_name("fairyc_fixture_zero_size_t")
unsafe extern C fn c::size_t zero_size_t() ;
@link_name("fairyc_fixture_four_size_t")
unsafe extern C fn c::size_t four_size_t() ;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn c::size_t eight_size_t() ;
@link_name("fy_std_buffer_initialize_v0")
unsafe extern C fn u32 initialize(c::size_t capacity, ptr mut fy_std_buffer_v0 output) ;
@link_name("fy_std_buffer_resize_zeroed_v0")
unsafe extern C fn u32 resize_zeroed(ptr mut fy_std_buffer_v0 buffer, c::size_t new_length) ;
@link_name("fairyc_fixture_buffer_is_zeroed")
unsafe extern C fn u32 buffer_is_zeroed(ptr fy_std_buffer_v0 buffer) ;
@link_name("fy_std_buffer_drop_v0")
unsafe extern C fn u32 drop_buffer(ptr mut fy_std_buffer_v0 buffer) ;
@link_name("fairyc_fixture_buffer_is_empty")
unsafe extern C fn u32 buffer_is_empty(ptr fy_std_buffer_v0 buffer) ;

fn i32 main() {
    let c::size_t zero = unsafe { zero_size_t() };
    let fy_std_system_heap_block_v0 empty_block = fy_std_system_heap_block_v0 {
        .data = null,
        .size = zero,
        .alignment = zero,
    };
    var fy_std_buffer_v0 buffer = fy_std_buffer_v0 {
        .block = empty_block,
        .length = zero,
    };
    let c::size_t capacity = unsafe { eight_size_t() };
    let u32 initialized = unsafe {
        initialize(capacity, ref mut buffer as ptr mut fy_std_buffer_v0)
    };
    let c::size_t length = unsafe { four_size_t() };
    let u32 resized = unsafe {
        resize_zeroed(ref mut buffer as ptr mut fy_std_buffer_v0, length)
    };
    let u32 observed = unsafe {
        buffer_is_zeroed(ref buffer as ptr fy_std_buffer_v0)
    };
    let u32 dropped = unsafe {
        drop_buffer(ref mut buffer as ptr mut fy_std_buffer_v0)
    };
    let u32 cleared = unsafe {
        buffer_is_empty(ref buffer as ptr fy_std_buffer_v0)
    };
    return if (initialized == 0 && resized == 0 && observed == 1 && dropped == 0 && cleared == 1) {
        42
    } else {
        1
    };
}
`

const bufferSourceCallBridge = `
size_t fairyc_fixture_zero_size_t(void) { return (size_t)0; }
size_t fairyc_fixture_four_size_t(void) { return (size_t)4; }
size_t fairyc_fixture_eight_size_t(void) { return (size_t)8; }

uint32_t fairyc_fixture_buffer_is_zeroed(const struct fy_std_buffer_v0 *buffer) {
    if (buffer == NULL || buffer->block.data == NULL ||
        buffer->block.size != (size_t)8 || buffer->block.alignment != (size_t)1 ||
        buffer->length != (size_t)4) {
        return UINT32_C(0);
    }
    const uint8_t *bytes = (const uint8_t *)buffer->block.data;
    return bytes[0] == UINT8_C(0) && bytes[1] == UINT8_C(0) &&
           bytes[2] == UINT8_C(0) && bytes[3] == UINT8_C(0) ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_buffer_is_empty(const struct fy_std_buffer_v0 *buffer) {
    return buffer != NULL && buffer->block.data == NULL &&
           buffer->block.size == (size_t)0 && buffer->block.alignment == (size_t)0 &&
           buffer->length == (size_t)0 ? UINT32_C(1) : UINT32_C(0);
}
`

var bufferSourceCallExpectedCFragments = []string{
	"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
	"struct fy_std_system_heap_block_v0 {",
	"void * data;",
	"size_t size;",
	"size_t alignment;",
	"struct fy_std_buffer_v0 {",
	"struct fy_std_system_heap_block_v0 block;",
	"size_t length;",
	"_Static_assert(sizeof(struct fy_std_buffer_v0) == UINT64_C(32)",
	"offsetof(struct fy_std_buffer_v0, length) == UINT64_C(24)",
	"extern uint32_t fy_std_buffer_initialize_v0(size_t fy_a_0, struct fy_std_buffer_v0 * fy_a_1);",
	"extern uint32_t fy_std_buffer_resize_zeroed_v0(struct fy_std_buffer_v0 * fy_a_0, size_t fy_a_1);",
	"extern uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 * fy_a_0);",
	" = &fy_l_",
}
