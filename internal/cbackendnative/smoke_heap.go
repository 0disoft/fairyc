package cbackendnative

const systemHeapSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_system_heap_block_v0 {
    ptr mut void data;
    c::size_t size;
    c::size_t alignment;
}

@link_name("fairyc_fixture_zero_size_t")
unsafe extern C fn c::size_t zero_size_t() ;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn c::size_t eight_size_t() ;
@link_name("fy_std_system_heap_allocate_v0")
unsafe extern C fn u32 allocate(c::size_t size, c::size_t alignment, ptr mut fy_std_system_heap_block_v0 output) ;
@link_name("fairyc_fixture_block_is_allocated")
unsafe extern C fn u32 block_is_allocated(ptr fy_std_system_heap_block_v0 block) ;
@link_name("fy_std_system_heap_deallocate_v0")
unsafe extern C fn u32 deallocate(ptr mut fy_std_system_heap_block_v0 block) ;
@link_name("fairyc_fixture_block_is_empty")
unsafe extern C fn u32 block_is_empty(ptr fy_std_system_heap_block_v0 block) ;

fn i32 main() {
    let c::size_t zero = unsafe { zero_size_t() };
    var fy_std_system_heap_block_v0 block = fy_std_system_heap_block_v0 {
        .data = null,
        .size = zero,
        .alignment = zero,
    };
    let c::size_t size = unsafe { eight_size_t() };
    let c::size_t alignment = unsafe { eight_size_t() };
    let u32 allocated = unsafe {
        allocate(size, alignment, ref mut block as ptr mut fy_std_system_heap_block_v0)
    };
    let u32 observed = unsafe {
        block_is_allocated(ref block as ptr fy_std_system_heap_block_v0)
    };
    let u32 deallocated = unsafe {
        deallocate(ref mut block as ptr mut fy_std_system_heap_block_v0)
    };
    let u32 cleared = unsafe {
        block_is_empty(ref block as ptr fy_std_system_heap_block_v0)
    };
    return if (allocated == 0 && observed == 1 && deallocated == 0 && cleared == 1) { 42 } else { 1 };
}
`

const systemHeapSourceCallBridge = `
size_t fairyc_fixture_zero_size_t(void) { return (size_t)0; }
size_t fairyc_fixture_eight_size_t(void) { return (size_t)8; }

uint32_t fairyc_fixture_block_is_allocated(const struct fy_std_system_heap_block_v0 *block) {
    return block != NULL && block->data != NULL && block->size == (size_t)8 &&
           block->alignment == (size_t)8 ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_block_is_empty(const struct fy_std_system_heap_block_v0 *block) {
    return block != NULL && block->data == NULL && block->size == (size_t)0 &&
           block->alignment == (size_t)0 ? UINT32_C(1) : UINT32_C(0);
}
`

var systemHeapSourceCallExpectedCFragments = []string{
	"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
	"struct fy_std_system_heap_block_v0 {",
	"void * data;",
	"size_t size;",
	"size_t alignment;",
	"extern uint32_t fy_std_system_heap_allocate_v0(size_t fy_a_0, size_t fy_a_1, struct fy_std_system_heap_block_v0 * fy_a_2);",
	"extern uint32_t fairyc_fixture_block_is_allocated(const struct fy_std_system_heap_block_v0 * fy_a_0);",
	"extern uint32_t fy_std_system_heap_deallocate_v0(struct fy_std_system_heap_block_v0 * fy_a_0);",
	" = &fy_l_",
}
