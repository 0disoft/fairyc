module examples::windows_target::main;

@link_name("fairyc_unused_windows_long")
unsafe extern C fn c::long unused_windows_long(c::long value) ;

@link_name("fy_std_text_utf8_valid_v0")
unsafe extern C fn u32 utf8_valid(ptr u8 data, usize length) ;

resource native_handle {
    ptr mut void raw;
    u32 state;

    cleanup {
        let u32 final_state = this.state;
    }
}

@repr(C)
struct system_heap_block {
    ptr mut void data;
    usize size;
    usize alignment;
}

@repr(C)
struct buffer {
    system_heap_block block;
    usize length;
}

@repr(C)
struct arena_node {
    system_heap_block storage;
    ptr mut arena_node next;
}

@repr(C)
struct arena {
    ptr mut arena_node head;
    usize generation;
    u8 active_region;
}

@repr(C)
struct arena_region {
    ptr mut arena owner;
    ptr mut arena_node mark;
    usize generation;
    u8 active;
}

@repr(C)
struct text_view {
    ptr u8 data;
    usize length;
}

@repr(C)
struct owned_string {
    buffer bytes;
}

@repr(C)
struct cstr_view {
    ptr u8 data;
    usize length;
}

@repr(C)
struct owned_cstring {
    buffer bytes;
}

@repr(C)
struct owned_text_result {
    u32 status;
    u32 allocation_status;
}

@link_name("fy_std_system_heap_allocate_v0")
unsafe extern C fn u32 allocate(usize size, usize alignment, ptr mut system_heap_block output) ;

@link_name("fy_std_system_heap_deallocate_v0")
unsafe extern C fn u32 deallocate(ptr mut system_heap_block block) ;

@link_name("fy_std_buffer_initialize_v0")
unsafe extern C fn u32 initialize_buffer(usize capacity, ptr mut buffer output) ;

@link_name("fy_std_buffer_resize_zeroed_v0")
unsafe extern C fn u32 resize_buffer_zeroed(ptr mut buffer value, usize new_length) ;

@link_name("fy_std_buffer_drop_v0")
unsafe extern C fn u32 drop_buffer(ptr mut buffer value) ;

@link_name("fy_std_arena_initialize_v0")
unsafe extern C fn u32 initialize_arena(ptr mut arena output) ;

@link_name("fy_std_region_begin_v0")
unsafe extern C fn u32 begin_region(ptr mut arena owner, ptr mut arena_region output) ;

@link_name("fy_std_region_allocate_zeroed_v0")
unsafe extern C fn u32 allocate_region_zeroed(ptr mut arena_region scope, usize size, usize alignment, ptr mut ptr mut void output) ;

@link_name("fy_std_region_reset_v0")
unsafe extern C fn u32 reset_region(ptr mut arena_region scope) ;

@link_name("fy_std_arena_drop_v0")
unsafe extern C fn u32 drop_arena(ptr mut arena value) ;

@link_name("fy_std_cstr_validate_v0")
unsafe extern C fn u32 validate_cstr(ptr u8 storage, usize storage_length, ptr mut cstr_view output) ;

@link_name("fy_std_string_copy_v0")
unsafe extern C fn owned_text_result copy_string(ptr text_view input, ptr mut owned_string output) ;

@link_name("fy_std_string_as_view_v0")
unsafe extern C fn owned_text_result view_string(ptr owned_string input, ptr mut text_view output) ;

@link_name("fy_std_string_append_v0")
unsafe extern C fn owned_text_result append_string(ptr mut owned_string value, ptr text_view suffix) ;

@link_name("fy_std_string_drop_v0")
unsafe extern C fn owned_text_result drop_string(ptr mut owned_string value) ;

@link_name("fy_std_cstring_from_text_v0")
unsafe extern C fn owned_text_result create_cstring(ptr text_view input, ptr mut owned_cstring output) ;

@link_name("fy_std_cstring_as_view_v0")
unsafe extern C fn owned_text_result view_cstring(ptr owned_cstring input, ptr mut cstr_view output) ;

@link_name("fy_std_cstring_drop_v0")
unsafe extern C fn owned_text_result drop_cstring(ptr mut owned_cstring value) ;

fn i32 main() {
    let native_handle handle = native_handle { .raw = null, .state = 42 };
    let bool handle_valid = handle.raw == null && handle.state == 42;
    var system_heap_block block = system_heap_block {
        .data = null,
        .size = 0,
        .alignment = 0,
    };
    let u32 allocated = unsafe {
        allocate(8, 8, ref mut block as ptr mut system_heap_block)
    };
    let bool allocation_valid = block.size == 8 && block.alignment == 8;
    let u32 deallocated = unsafe {
        deallocate(ref mut block as ptr mut system_heap_block)
    };
    let bool deallocation_valid = block.size == 0 && block.alignment == 0;
    let system_heap_block empty_block = system_heap_block {
        .data = null,
        .size = 0,
        .alignment = 0,
    };
    var buffer bytes = buffer {
        .block = empty_block,
        .length = 0,
    };
    let u32 initialized = unsafe {
        initialize_buffer(8, ref mut bytes as ptr mut buffer)
    };
    let u32 resized = unsafe {
        resize_buffer_zeroed(ref mut bytes as ptr mut buffer, 4)
    };
    let bool buffer_valid = bytes.block.size == 8 && bytes.block.alignment == 1 &&
        bytes.length == 4;
    let u32 dropped = unsafe {
        drop_buffer(ref mut bytes as ptr mut buffer)
    };
    let bool buffer_dropped = bytes.block.size == 0 && bytes.block.alignment == 0 &&
        bytes.length == 0;
    var arena storage = arena {
        .head = null,
        .generation = 0,
        .active_region = 0,
    };
    var arena_region scope = arena_region {
        .owner = null,
        .mark = null,
        .generation = 0,
        .active = 0,
    };
    let u32 arena_initialized = unsafe {
        initialize_arena(ref mut storage as ptr mut arena)
    };
    let bool arena_initial_state = storage.head == null && storage.generation == 1 &&
        storage.active_region == 0;
    let u32 region_started = unsafe {
        begin_region(ref mut storage as ptr mut arena, ref mut scope as ptr mut arena_region)
    };
    let bool region_active = storage.active_region == 1 && scope.owner != null &&
        scope.mark == null && scope.generation == 1 && scope.active == 1;
    var ptr mut void region_data = null;
    let u32 region_allocated = unsafe {
        allocate_region_zeroed(
            ref mut scope as ptr mut arena_region,
            8,
            8,
            ref mut region_data as ptr mut ptr mut void
        )
    };
    let bool region_allocation_state = storage.head != null && region_data != null &&
        storage.generation == 1 && storage.active_region == 1;
    let u32 region_reset = unsafe {
        reset_region(ref mut scope as ptr mut arena_region)
    };
    let bool region_reset_state = storage.head == null && storage.generation == 2 &&
        storage.active_region == 0 && scope.owner == null && scope.mark == null &&
        scope.generation == 0 && scope.active == 0;
    let u32 arena_dropped = unsafe {
        drop_arena(ref mut storage as ptr mut arena)
    };
    let bool arena_drop_state = storage.head == null && storage.generation == 3 &&
        storage.active_region == 0;
    let ref str text = "Fairy";
    let text_view input = text_view {
        .data = unsafe { text as ptr u8 },
        .length = 5,
    };
    let buffer empty_string_buffer = buffer {
        .block = system_heap_block {
            .data = null,
            .size = 0,
            .alignment = 0,
        },
        .length = 0,
    };
    var owned_string string_value = owned_string { .bytes = empty_string_buffer };
    var text_view string_view = text_view {
        .data = null,
        .length = 0,
    };
    let owned_text_result string_copied = unsafe {
        copy_string(ref input as ptr text_view, ref mut string_value as ptr mut owned_string)
    };
    let bool string_copy_state = string_value.bytes.block.data != null &&
        string_value.bytes.block.alignment == 1 && string_value.bytes.length == 5;
    let owned_text_result string_viewed_once = unsafe {
        view_string(ref string_value as ptr owned_string, ref mut string_view as ptr mut text_view)
    };
    let bool string_view_once_state = string_view.data != null && string_view.length == 5;
    let owned_text_result string_appended = unsafe {
        append_string(ref mut string_value as ptr mut owned_string, ref input as ptr text_view)
    };
    let bool string_append_state = string_value.bytes.block.data != null &&
        string_value.bytes.length == 10;
    let owned_text_result string_viewed_twice = unsafe {
        view_string(ref string_value as ptr owned_string, ref mut string_view as ptr mut text_view)
    };
    let bool string_view_twice_state = string_view.data != null && string_view.length == 10;
    let owned_text_result string_dropped = unsafe {
        drop_string(ref mut string_value as ptr mut owned_string)
    };
    let bool string_drop_state = string_value.bytes.block.data == null &&
        string_value.bytes.block.size == 0 && string_value.bytes.block.alignment == 0 &&
        string_value.bytes.length == 0;
    let buffer empty_cstring_buffer = buffer {
        .block = system_heap_block {
            .data = null,
            .size = 0,
            .alignment = 0,
        },
        .length = 0,
    };
    var owned_cstring cstring_value = owned_cstring { .bytes = empty_cstring_buffer };
    var cstr_view cstring_view = cstr_view {
        .data = null,
        .length = 0,
    };
    let owned_text_result cstring_created = unsafe {
        create_cstring(ref input as ptr text_view, ref mut cstring_value as ptr mut owned_cstring)
    };
    let bool cstring_create_state = cstring_value.bytes.block.data != null &&
        cstring_value.bytes.block.alignment == 1 && cstring_value.bytes.length == 6;
    let owned_text_result cstring_viewed = unsafe {
        view_cstring(ref cstring_value as ptr owned_cstring, ref mut cstring_view as ptr mut cstr_view)
    };
    let bool cstring_view_state = cstring_view.data != null && cstring_view.length == 5;
    let owned_text_result cstring_dropped = unsafe {
        drop_cstring(ref mut cstring_value as ptr mut owned_cstring)
    };
    let bool cstring_drop_state = cstring_value.bytes.block.data == null &&
        cstring_value.bytes.block.size == 0 && cstring_value.bytes.block.alignment == 0 &&
        cstring_value.bytes.length == 0;
    var cstr_view direct_cstr_view = cstr_view {
        .data = null,
        .length = 0,
    };
    let u32 direct_cstr_validated = unsafe {
        validate_cstr(c"Fairy" as ptr u8, 6, ref mut direct_cstr_view as ptr mut cstr_view)
    };
    let bool direct_cstr_state = direct_cstr_view.data != null && direct_cstr_view.length == 5;
    return if (handle_valid && allocated == 0 && allocation_valid &&
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
        unsafe { utf8_valid(null, 0) } == 1) {
        42
    } else {
        1
    };
}
