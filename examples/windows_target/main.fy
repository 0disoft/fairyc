module examples::windows_target::main;

@link_name("fairyc_unused_windows_long")
unsafe extern C fn unused_windows_long(value: c::long) -> c::long;

@link_name("fy_std_text_utf8_valid_v0")
unsafe extern C fn utf8_valid(data: *u8, length: usize) -> u32;

resource native_handle {
    raw: *mut void;
    state: u32;

    drop fn(self: &mut Self) {
        let final_state: u32 = self.state;
    }
}

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

@repr(C)
struct text_view {
    data: *u8;
    length: usize;
}

@repr(C)
struct owned_string {
    bytes: buffer;
}

@repr(C)
struct cstr_view {
    data: *u8;
    length: usize;
}

@repr(C)
struct owned_cstring {
    bytes: buffer;
}

@repr(C)
struct owned_text_result {
    status: u32;
    allocation_status: u32;
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

@link_name("fy_std_cstr_validate_v0")
unsafe extern C fn validate_cstr(
    storage: *u8,
    storage_length: usize,
    output: *mut cstr_view
) -> u32;

@link_name("fy_std_string_copy_v0")
unsafe extern C fn copy_string(
    input: *text_view,
    output: *mut owned_string
) -> owned_text_result;

@link_name("fy_std_string_as_view_v0")
unsafe extern C fn view_string(
    input: *owned_string,
    output: *mut text_view
) -> owned_text_result;

@link_name("fy_std_string_append_v0")
unsafe extern C fn append_string(
    value: *mut owned_string,
    suffix: *text_view
) -> owned_text_result;

@link_name("fy_std_string_drop_v0")
unsafe extern C fn drop_string(value: *mut owned_string) -> owned_text_result;

@link_name("fy_std_cstring_from_text_v0")
unsafe extern C fn create_cstring(
    input: *text_view,
    output: *mut owned_cstring
) -> owned_text_result;

@link_name("fy_std_cstring_as_view_v0")
unsafe extern C fn view_cstring(
    input: *owned_cstring,
    output: *mut cstr_view
) -> owned_text_result;

@link_name("fy_std_cstring_drop_v0")
unsafe extern C fn drop_cstring(value: *mut owned_cstring) -> owned_text_result;

fn main() -> i32 {
    let handle: native_handle = native_handle { raw: null, state: 42 };
    let handle_valid: bool = handle.raw == null && handle.state == 42;
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
    let text: &str = "Fairy";
    let input: text_view = text_view {
        data: unsafe { text as *u8 },
        length: 5,
    };
    let empty_string_buffer: buffer = buffer {
        block: system_heap_block {
            data: null,
            size: 0,
            alignment: 0,
        },
        length: 0,
    };
    var string_value: owned_string = owned_string { bytes: empty_string_buffer };
    var string_view: text_view = text_view {
        data: null,
        length: 0,
    };
    let string_copied: owned_text_result = unsafe {
        copy_string(&input as *text_view, &mut string_value as *mut owned_string)
    };
    let string_copy_state: bool = string_value.bytes.block.data != null &&
        string_value.bytes.block.alignment == 1 && string_value.bytes.length == 5;
    let string_viewed_once: owned_text_result = unsafe {
        view_string(&string_value as *owned_string, &mut string_view as *mut text_view)
    };
    let string_view_once_state: bool = string_view.data != null && string_view.length == 5;
    let string_appended: owned_text_result = unsafe {
        append_string(&mut string_value as *mut owned_string, &input as *text_view)
    };
    let string_append_state: bool = string_value.bytes.block.data != null &&
        string_value.bytes.length == 10;
    let string_viewed_twice: owned_text_result = unsafe {
        view_string(&string_value as *owned_string, &mut string_view as *mut text_view)
    };
    let string_view_twice_state: bool = string_view.data != null && string_view.length == 10;
    let string_dropped: owned_text_result = unsafe {
        drop_string(&mut string_value as *mut owned_string)
    };
    let string_drop_state: bool = string_value.bytes.block.data == null &&
        string_value.bytes.block.size == 0 && string_value.bytes.block.alignment == 0 &&
        string_value.bytes.length == 0;
    let empty_cstring_buffer: buffer = buffer {
        block: system_heap_block {
            data: null,
            size: 0,
            alignment: 0,
        },
        length: 0,
    };
    var cstring_value: owned_cstring = owned_cstring { bytes: empty_cstring_buffer };
    var cstring_view: cstr_view = cstr_view {
        data: null,
        length: 0,
    };
    let cstring_created: owned_text_result = unsafe {
        create_cstring(&input as *text_view, &mut cstring_value as *mut owned_cstring)
    };
    let cstring_create_state: bool = cstring_value.bytes.block.data != null &&
        cstring_value.bytes.block.alignment == 1 && cstring_value.bytes.length == 6;
    let cstring_viewed: owned_text_result = unsafe {
        view_cstring(&cstring_value as *owned_cstring, &mut cstring_view as *mut cstr_view)
    };
    let cstring_view_state: bool = cstring_view.data != null && cstring_view.length == 5;
    let cstring_dropped: owned_text_result = unsafe {
        drop_cstring(&mut cstring_value as *mut owned_cstring)
    };
    let cstring_drop_state: bool = cstring_value.bytes.block.data == null &&
        cstring_value.bytes.block.size == 0 && cstring_value.bytes.block.alignment == 0 &&
        cstring_value.bytes.length == 0;
    var direct_cstr_view: cstr_view = cstr_view {
        data: null,
        length: 0,
    };
    let direct_cstr_validated: u32 = unsafe {
        validate_cstr(c"Fairy" as *u8, 6, &mut direct_cstr_view as *mut cstr_view)
    };
    let direct_cstr_state: bool = direct_cstr_view.data != null && direct_cstr_view.length == 5;
    if handle_valid && allocated == 0 && allocation_valid &&
        deallocated == 0 && deallocation_valid &&
        initialized == 0 && resized == 0 && buffer_valid && dropped == 0 && buffer_dropped &&
        arena_initialized == 0 && arena_initial_state && region_started == 0 && region_active &&
        region_allocated == 0 && region_allocation_state && region_reset == 0 &&
        region_reset_state && arena_dropped == 0 && arena_drop_state &&
        string_copied.status == 0 && string_copied.allocation_status == 0 && string_copy_state &&
        string_viewed_once.status == 0 && string_viewed_once.allocation_status == 0 &&
        string_view_once_state && string_appended.status == 0 &&
        string_appended.allocation_status == 0 && string_append_state &&
        string_viewed_twice.status == 0 && string_viewed_twice.allocation_status == 0 &&
        string_view_twice_state && string_dropped.status == 0 &&
        string_dropped.allocation_status == 0 && string_drop_state &&
        cstring_created.status == 0 && cstring_created.allocation_status == 0 &&
        cstring_create_state && cstring_viewed.status == 0 &&
        cstring_viewed.allocation_status == 0 && cstring_view_state &&
        cstring_dropped.status == 0 && cstring_dropped.allocation_status == 0 &&
        cstring_drop_state && direct_cstr_validated == 0 && direct_cstr_state &&
        unsafe { utf8_valid(null, 0) } == 1 {
        42
    } else {
        1
    }
}
