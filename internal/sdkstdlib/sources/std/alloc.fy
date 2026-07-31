module std::alloc;

/// Owns bulk-allocation storage used only through lexical Region capabilities.
pub resource Arena {
    ptr mut void head;
    usize generation;
    u8 active_region;

    cleanup {
        let u32 dropped = unsafe {
            arena_drop_raw(ref mut this as ptr mut void)
        };
    }
}

/// Reports the stable allocation status returned by the target runtime.
pub struct AllocError {
    u32 status;
}

@link_name("fy_std_arena_initialize_v0")
unsafe extern C fn u32 arena_initialize_raw(ptr mut void output) ;

@link_name("fy_std_arena_drop_v0")
unsafe extern C fn u32 arena_drop_raw(ptr mut void arena) ;

fn Arena empty_arena() {
    return Arena {
        .head = null,
        .generation = 0,
        .active_region = 0,
    };
}

fn AllocError alloc_error(u32 status) {
    return AllocError { .status = status };
}

fn i32 discard_arena(Arena value) {
    return 0;
}

/// Creates an empty Arena backed by the target SystemHeap implementation.
pub fn Result<Arena, AllocError> create() {
    var Arena output = empty_arena();
    let u32 status = unsafe {
        arena_initialize_raw(ref mut output as ptr mut void)
    };
    return if (status == 0) {
        .ok(output)
    } else {
        let i32 discarded = discard_arena(output);
        .err(alloc_error(status))
    };
}

/// Returns the stable allocation status code carried by error.
pub fn u32 error_status(AllocError error) {
    return error.status;
}
