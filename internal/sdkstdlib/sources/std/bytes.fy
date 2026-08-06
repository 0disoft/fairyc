module std::bytes;

/// Owns a contiguous byte allocation and releases it automatically at scope exit.
pub resource Buffer {
    ptr mut void data;
    usize capacity;
    usize alignment;
    usize length;

    cleanup {
        let u32 dropped = unsafe {
            buffer_drop_raw(ref mut this as ptr mut void)
        };
    }
}

/// Reports the stable allocation status returned by the target runtime.
pub struct BufferError {
    u32 status;
}

@link_name("fy_std_buffer_initialize_v0")
unsafe extern C fn u32 buffer_initialize_raw(usize capacity, ptr mut void output) ;

@link_name("fy_std_buffer_reserve_v0")
unsafe extern C fn u32 buffer_reserve_raw(ptr mut void buffer, usize minimum_capacity) ;

@link_name("fy_std_buffer_append_v0")
unsafe extern C fn u32 buffer_append_raw(ptr mut void buffer, ptr u8 source, usize count) ;

@link_name("fy_std_buffer_resize_zeroed_v0")
unsafe extern C fn u32 buffer_resize_zeroed_raw(ptr mut void buffer, usize new_length) ;

@link_name("fy_std_buffer_truncate_v0")
unsafe extern C fn u32 buffer_truncate_raw(ptr mut void buffer, usize new_length) ;

@link_name("fy_std_buffer_clear_v0")
unsafe extern C fn u32 buffer_clear_raw(ptr mut void buffer) ;

@link_name("fy_std_buffer_drop_v0")
unsafe extern C fn u32 buffer_drop_raw(ptr mut void buffer) ;

fn Buffer empty_buffer() {
    return Buffer {
        .data = null,
        .capacity = 0,
        .alignment = 0,
        .length = 0,
    };
}

fn BufferError buffer_error(u32 status) {
    return BufferError { .status = status };
}

fn i32 discard_buffer(Buffer value) {
    return 0;
}

/// Creates an empty buffer with at least the requested capacity.
pub fn Result<Buffer, BufferError> with_capacity(usize capacity) {
    var Buffer output = empty_buffer();
    let u32 status = unsafe {
        buffer_initialize_raw(capacity, ref mut output as ptr mut void)
    };
    return if (status == 0) {
        .ok(output)
    } else {
        let i32 discarded = discard_buffer(output);
        .err(buffer_error(status))
    };
}

/// Copies the exact bytes in source into a newly owned buffer.
pub fn Result<Buffer, BufferError> copy(ref u8[] source) {
    var Buffer output = empty_buffer();
    let u32 initialized = unsafe {
        buffer_initialize_raw(source.len, ref mut output as ptr mut void)
    };
    return if (initialized != 0) {
        let i32 discarded = discard_buffer(output);
        .err(buffer_error(initialized))
    } else {
        let u32 appended = unsafe {
            buffer_append_raw(ref mut output as ptr mut void, source as ptr u8, source.len)
        };
        if (appended == 0) {
            .ok(output)
        } else {
            let i32 discarded = discard_buffer(output);
            .err(buffer_error(appended))
        }
    };
}

/// Returns the initialized byte count.
pub fn usize len(ref Buffer buffer) {
    return buffer.length;
}

/// Returns the allocation capacity in bytes.
pub fn usize capacity(ref Buffer buffer) {
    return buffer.capacity;
}

/// Reports whether the initialized byte count is zero.
pub fn bool is_empty(ref Buffer buffer) {
    return buffer.length == 0;
}

/// Borrows the initialized prefix for read-only access.
pub fn ref u8[] view(ref Buffer buffer) {
    return unsafe {
        byte_slice::from_raw_parts(buffer, buffer.data, buffer.length)
    };
}

/// Borrows the initialized prefix for exclusive mutation without changing its length.
pub fn ref mut u8[] view_mut(ref mut Buffer buffer) {
    return unsafe {
        byte_slice::from_raw_parts_mut(buffer, buffer.data, buffer.length)
    };
}

/// Ensures capacity is at least minimum_capacity and returns the resulting capacity.
pub fn Result<usize, BufferError> reserve(ref mut Buffer buffer, usize minimum_capacity) {
    let u32 status = unsafe {
        buffer_reserve_raw(buffer as ptr mut void, minimum_capacity)
    };
    return if (status == 0) {
        .ok(buffer.capacity)
    } else {
        .err(buffer_error(status))
    };
}

/// Appends source exactly and returns the new initialized byte count.
pub fn Result<usize, BufferError> append(ref mut Buffer buffer, ref u8[] source) {
    let u32 status = unsafe {
        buffer_append_raw(buffer as ptr mut void, source as ptr u8, source.len)
    };
    return if (status == 0) {
        .ok(buffer.length)
    } else {
        .err(buffer_error(status))
    };
}

/// Appends one byte and returns the new initialized byte count.
pub fn Result<usize, BufferError> append_byte(ref mut Buffer buffer, u8 value) {
    let u32 status = unsafe {
        buffer_append_raw(buffer as ptr mut void, ref value as ptr u8, 1)
    };
    return if (status == 0) {
        .ok(buffer.length)
    } else {
        .err(buffer_error(status))
    };
}

/// Changes the initialized byte count, zero-filling every newly exposed byte.
pub fn Result<usize, BufferError> resize_zeroed(ref mut Buffer buffer, usize new_length) {
    let u32 status = unsafe {
        buffer_resize_zeroed_raw(buffer as ptr mut void, new_length)
    };
    return if (status == 0) {
        .ok(buffer.length)
    } else {
        .err(buffer_error(status))
    };
}

/// Shrinks the initialized prefix and returns the new byte count.
pub fn Result<usize, BufferError> truncate(ref mut Buffer buffer, usize new_length) {
    let u32 status = unsafe {
        buffer_truncate_raw(buffer as ptr mut void, new_length)
    };
    return if (status == 0) {
        .ok(buffer.length)
    } else {
        .err(buffer_error(status))
    };
}

/// Sets the initialized byte count to zero while retaining the allocation.
pub fn Result<usize, BufferError> clear(ref mut Buffer buffer) {
    let u32 status = unsafe {
        buffer_clear_raw(buffer as ptr mut void)
    };
    return if (status == 0) {
        .ok(buffer.length)
    } else {
        .err(buffer_error(status))
    };
}

/// Returns the stable allocation status code carried by error.
pub fn u32 error_status(BufferError error) {
    return error.status;
}
