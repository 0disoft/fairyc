package cbackendnative

const arenaSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_arena_v0 {
    ptr mut fy_std_arena_node_v0 head;
    c::ulong generation;
    u8 active_region;
}
@repr(C)
struct fy_std_region_v0 {
    ptr mut fy_std_arena_v0 arena;
    ptr mut fy_std_arena_node_v0 mark;
    c::ulong generation;
    u8 active;
}
@repr(C)
struct fy_std_arena_node_v0 {
    fy_std_system_heap_block_v0 storage;
    ptr mut fy_std_arena_node_v0 next;
}
@repr(C)
struct fy_std_system_heap_block_v0 {
    ptr mut void data;
    c::size_t size;
    c::size_t alignment;
}

@link_name("fairyc_fixture_zero_c_ulong")
unsafe extern C fn c::ulong zero_c_ulong() ;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn c::size_t eight_size_t() ;
@link_name("fy_std_arena_initialize_v0")
unsafe extern C fn u32 initialize_arena(ptr mut fy_std_arena_v0 output) ;
@link_name("fy_std_region_begin_v0")
unsafe extern C fn u32 begin_region(ptr mut fy_std_arena_v0 arena, ptr mut fy_std_region_v0 output) ;
@link_name("fy_std_region_allocate_zeroed_v0")
unsafe extern C fn u32 allocate_zeroed(ptr mut fy_std_region_v0 scope, c::size_t size, c::size_t alignment, ptr mut ptr mut void output) ;
@link_name("fairyc_fixture_arena_allocation_is_zeroed")
unsafe extern C fn u32 allocation_is_zeroed(ptr mut fy_std_arena_v0 arena, ptr mut fy_std_region_v0 scope, ptr mut void data) ;
@link_name("fy_std_region_reset_v0")
unsafe extern C fn u32 reset_region(ptr mut fy_std_region_v0 scope) ;
@link_name("fairyc_fixture_arena_region_is_reset")
unsafe extern C fn u32 region_is_reset(ptr mut fy_std_arena_v0 arena, ptr mut fy_std_region_v0 scope) ;
@link_name("fy_std_arena_drop_v0")
unsafe extern C fn u32 drop_arena(ptr mut fy_std_arena_v0 arena) ;
@link_name("fairyc_fixture_arena_is_dropped")
unsafe extern C fn u32 arena_is_dropped(ptr mut fy_std_arena_v0 arena) ;

fn i32 main() {
    let c::ulong zero_generation = unsafe { zero_c_ulong() };
    var fy_std_arena_v0 arena = fy_std_arena_v0 {
        .head = null,
        .generation = zero_generation,
        .active_region = 0,
    };
    var fy_std_region_v0 scope = fy_std_region_v0 {
        .arena = null,
        .mark = null,
        .generation = zero_generation,
        .active = 0,
    };
    let u32 initialized = unsafe {
        initialize_arena(ref mut arena as ptr mut fy_std_arena_v0)
    };
    let u32 began = unsafe {
        begin_region(
            ref mut arena as ptr mut fy_std_arena_v0,
            ref mut scope as ptr mut fy_std_region_v0
        )
    };
    let c::size_t size = unsafe { eight_size_t() };
    let c::size_t alignment = unsafe { eight_size_t() };
    var ptr mut void data = null;
    let u32 allocated = unsafe {
        allocate_zeroed(
            ref mut scope as ptr mut fy_std_region_v0,
            size,
            alignment,
            ref mut data as ptr mut ptr mut void
        )
    };
    let u32 observed = unsafe {
        allocation_is_zeroed(
            ref mut arena as ptr mut fy_std_arena_v0,
            ref mut scope as ptr mut fy_std_region_v0,
            data
        )
    };
    let u32 reset = unsafe {
        reset_region(ref mut scope as ptr mut fy_std_region_v0)
    };
    let u32 cleared = unsafe {
        region_is_reset(
            ref mut arena as ptr mut fy_std_arena_v0,
            ref mut scope as ptr mut fy_std_region_v0
        )
    };
    let u32 dropped = unsafe {
        drop_arena(ref mut arena as ptr mut fy_std_arena_v0)
    };
    let u32 finalized = unsafe {
        arena_is_dropped(ref mut arena as ptr mut fy_std_arena_v0)
    };
    return if (initialized == 0 && began == 0 && allocated == 0 && observed == 1 &&
        reset == 0 && cleared == 1 && dropped == 0 && finalized == 1) {
        42
    } else {
        1
    };
}
`

const arenaSourceCallBridge = `
_Static_assert(
    _Generic((uint64_t)0, unsigned long: 1, default: 0),
    "fairyc target uint64_t is unsigned long"
);

unsigned long fairyc_fixture_zero_c_ulong(void) { return 0UL; }
size_t fairyc_fixture_eight_size_t(void) { return (size_t)8; }

uint32_t fairyc_fixture_arena_allocation_is_zeroed(
    struct fy_std_arena_v0 *arena,
    struct fy_std_region_v0 *region,
    void *data
) {
    if (arena == NULL || region == NULL || data == NULL || arena->head == NULL ||
        arena->generation != UINT64_C(1) || arena->active_region != UINT8_C(1) ||
        region->arena != arena || region->mark != NULL ||
        region->generation != UINT64_C(1) || region->active != UINT8_C(1) ||
        (uintptr_t)data % (uintptr_t)8 != (uintptr_t)0) {
        return UINT32_C(0);
    }
    const uint8_t *bytes = (const uint8_t *)data;
    for (size_t index = 0; index < (size_t)8; ++index) {
        if (bytes[index] != UINT8_C(0)) {
            return UINT32_C(0);
        }
    }
    return UINT32_C(1);
}

uint32_t fairyc_fixture_arena_region_is_reset(
    struct fy_std_arena_v0 *arena,
    struct fy_std_region_v0 *region
) {
    return arena != NULL && region != NULL && arena->head == NULL &&
           arena->generation == UINT64_C(2) && arena->active_region == UINT8_C(0) &&
           region->arena == NULL && region->mark == NULL &&
           region->generation == UINT64_C(0) && region->active == UINT8_C(0) ?
           UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_arena_is_dropped(struct fy_std_arena_v0 *arena) {
    return arena != NULL && arena->head == NULL && arena->generation == UINT64_C(3) &&
           arena->active_region == UINT8_C(0) ? UINT32_C(1) : UINT32_C(0);
}
`

var arenaSourceCallExpectedCFragments = []string{
	"_Static_assert(sizeof(unsigned long) == UINT64_C(8), \"fairyc target c.ulong size\");",
	"struct fy_std_system_heap_block_v0 {",
	"struct fy_std_arena_node_v0 {",
	"struct fy_std_system_heap_block_v0 storage;",
	"struct fy_std_arena_node_v0 * next;",
	"_Static_assert(sizeof(struct fy_std_arena_node_v0) == UINT64_C(32)",
	"_Static_assert(_Alignof(struct fy_std_arena_node_v0) == UINT64_C(8)",
	"offsetof(struct fy_std_arena_node_v0, storage) == UINT64_C(0)",
	"offsetof(struct fy_std_arena_node_v0, next) == UINT64_C(24)",
	"struct fy_std_arena_v0 {",
	"struct fy_std_arena_node_v0 * head;",
	"unsigned long generation;",
	"uint8_t active_region;",
	"_Static_assert(sizeof(struct fy_std_arena_v0) == UINT64_C(24)",
	"offsetof(struct fy_std_arena_v0, head) == UINT64_C(0)",
	"offsetof(struct fy_std_arena_v0, generation) == UINT64_C(8)",
	"offsetof(struct fy_std_arena_v0, active_region) == UINT64_C(16)",
	"struct fy_std_region_v0 {",
	"struct fy_std_arena_v0 * arena;",
	"struct fy_std_arena_node_v0 * mark;",
	"_Static_assert(sizeof(struct fy_std_region_v0) == UINT64_C(32)",
	"offsetof(struct fy_std_region_v0, arena) == UINT64_C(0)",
	"offsetof(struct fy_std_region_v0, mark) == UINT64_C(8)",
	"offsetof(struct fy_std_region_v0, generation) == UINT64_C(16)",
	"offsetof(struct fy_std_region_v0, active) == UINT64_C(24)",
	"extern uint32_t fy_std_arena_initialize_v0(struct fy_std_arena_v0 * fy_a_0);",
	"extern uint32_t fy_std_region_begin_v0(struct fy_std_arena_v0 * fy_a_0, struct fy_std_region_v0 * fy_a_1);",
	"extern uint32_t fy_std_region_allocate_zeroed_v0(struct fy_std_region_v0 * fy_a_0, size_t fy_a_1, size_t fy_a_2, void * * fy_a_3);",
	"extern uint32_t fy_std_region_reset_v0(struct fy_std_region_v0 * fy_a_0);",
	"extern uint32_t fy_std_arena_drop_v0(struct fy_std_arena_v0 * fy_a_0);",
	" = &fy_l_",
}
