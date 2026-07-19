module examples::windows_target::main;

@link_name("fairyc_unused_windows_long")
unsafe extern C fn unused_windows_long(value: c::long) -> c::long;

@link_name("fy_std_text_utf8_valid_v0")
unsafe extern C fn utf8_valid(data: *u8, length: usize) -> u32;

@repr(C)
struct system_heap_block {
    data: *mut void;
    size: usize;
    alignment: usize;
}

@repr(C)
struct buffer {
    block: system_heap_block;
    length: usize;
}

@repr(C)
struct arena_node {
    storage: system_heap_block;
    next: *mut arena_node;
}

@repr(C)
struct arena {
    head: *mut arena_node;
    generation: usize;
    active_region: u8;
}

@repr(C)
struct arena_region {
    owner: *mut arena;
    mark: *mut arena_node;
    generation: usize;
    active: u8;
}

@link_name("fy_std_system_heap_allocate_v0")
unsafe extern C fn allocate(
    size: usize,
    alignment: usize,
    output: *mut system_heap_block
) -> u32;

@link_name("fy_std_system_heap_deallocate_v0")
unsafe extern C fn deallocate(block: *mut system_heap_block) -> u32;

@link_name("fy_std_buffer_initialize_v0")
unsafe extern C fn initialize_buffer(capacity: usize, output: *mut buffer) -> u32;

@link_name("fy_std_buffer_resize_zeroed_v0")
unsafe extern C fn resize_buffer_zeroed(value: *mut buffer, new_length: usize) -> u32;

@link_name("fy_std_buffer_drop_v0")
unsafe extern C fn drop_buffer(value: *mut buffer) -> u32;

@link_name("fy_std_arena_initialize_v0")
unsafe extern C fn initialize_arena(output: *mut arena) -> u32;

@link_name("fy_std_region_begin_v0")
unsafe extern C fn begin_region(owner: *mut arena, output: *mut arena_region) -> u32;

@link_name("fy_std_region_allocate_zeroed_v0")
unsafe extern C fn allocate_region_zeroed(
    scope: *mut arena_region,
    size: usize,
    alignment: usize,
    output: *mut *mut void
) -> u32;

@link_name("fy_std_region_reset_v0")
unsafe extern C fn reset_region(scope: *mut arena_region) -> u32;

@link_name("fy_std_arena_drop_v0")
unsafe extern C fn drop_arena(value: *mut arena) -> u32;

fn main() -> i32 {
    var block: system_heap_block = system_heap_block {
        data: null,
        size: 0,
        alignment: 0,
    };
    let allocated: u32 = unsafe {
        allocate(8, 8, &mut block as *mut system_heap_block)
    };
    let allocation_valid: bool = block.size == 8 && block.alignment == 8;
    let deallocated: u32 = unsafe {
        deallocate(&mut block as *mut system_heap_block)
    };
    let deallocation_valid: bool = block.size == 0 && block.alignment == 0;
    let empty_block: system_heap_block = system_heap_block {
        data: null,
        size: 0,
        alignment: 0,
    };
    var bytes: buffer = buffer {
        block: empty_block,
        length: 0,
    };
    let initialized: u32 = unsafe {
        initialize_buffer(8, &mut bytes as *mut buffer)
    };
    let resized: u32 = unsafe {
        resize_buffer_zeroed(&mut bytes as *mut buffer, 4)
    };
    let buffer_valid: bool = bytes.block.size == 8 && bytes.block.alignment == 1 &&
        bytes.length == 4;
    let dropped: u32 = unsafe {
        drop_buffer(&mut bytes as *mut buffer)
    };
    let buffer_dropped: bool = bytes.block.size == 0 && bytes.block.alignment == 0 &&
        bytes.length == 0;
    var storage: arena = arena {
        head: null,
        generation: 0,
        active_region: 0,
    };
    var scope: arena_region = arena_region {
        owner: null,
        mark: null,
        generation: 0,
        active: 0,
    };
    let arena_initialized: u32 = unsafe {
        initialize_arena(&mut storage as *mut arena)
    };
    let arena_initial_state: bool = storage.head == null && storage.generation == 1 &&
        storage.active_region == 0;
    let region_started: u32 = unsafe {
        begin_region(&mut storage as *mut arena, &mut scope as *mut arena_region)
    };
    let region_active: bool = storage.active_region == 1 && scope.owner != null &&
        scope.mark == null && scope.generation == 1 && scope.active == 1;
    var region_data: *mut void = null;
    let region_allocated: u32 = unsafe {
        allocate_region_zeroed(
            &mut scope as *mut arena_region,
            8,
            8,
            &mut region_data as *mut *mut void
        )
    };
    let region_allocation_state: bool = storage.head != null && region_data != null &&
        storage.generation == 1 && storage.active_region == 1;
    let region_reset: u32 = unsafe {
        reset_region(&mut scope as *mut arena_region)
    };
    let region_reset_state: bool = storage.head == null && storage.generation == 2 &&
        storage.active_region == 0 && scope.owner == null && scope.mark == null &&
        scope.generation == 0 && scope.active == 0;
    let arena_dropped: u32 = unsafe {
        drop_arena(&mut storage as *mut arena)
    };
    let arena_drop_state: bool = storage.head == null && storage.generation == 3 &&
        storage.active_region == 0;
    if allocated == 0 && allocation_valid && deallocated == 0 && deallocation_valid &&
        initialized == 0 && resized == 0 && buffer_valid && dropped == 0 && buffer_dropped &&
        arena_initialized == 0 && arena_initial_state && region_started == 0 && region_active &&
        region_allocated == 0 && region_allocation_state && region_reset == 0 &&
        region_reset_state && arena_dropped == 0 && arena_drop_state &&
        unsafe { utf8_valid(null, 0) } == 1 {
        42
    } else {
        1
    }
}
