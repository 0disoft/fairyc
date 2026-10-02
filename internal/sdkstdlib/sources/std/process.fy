module std::process;

@repr(C)
struct raw_process_args_result {
    u32 status;
    usize needed;
}

/// Reports the raw failure status and the exact UTF-8 byte length required for success.
pub struct ProcessArgError {
    u32 status;
    usize needed;
}

@link_name("fy_std_process_arg_count_v0")
unsafe extern C fn usize process_arg_count_raw() ;

@link_name("fy_std_process_arg_v0")
unsafe extern C fn raw_process_args_result process_arg_raw(usize index, ptr mut u8 data, usize capacity) ;

fn Result<usize, ProcessArgError> checked_arg_result(u32 status, usize needed) {
    return if (status == 0) {
        .ok(needed)
    } else {
        .err(ProcessArgError {
            .status = status,
            .needed = needed,
        })
    };
}

/// Returns the number of process arguments, including the executable name at index 0.
pub fn usize arg_count() {
    return unsafe {
        process_arg_count_raw()
    };
}

/// Copies the UTF-8 bytes of one process argument into buffer and returns their exact length.
/// Status 1 is index out of range, status 2 is invalid or unavailable Unicode argv, and
/// status 3 is buffer too small with `needed` carrying the full UTF-8 byte length.
pub fn Result<usize, ProcessArgError> arg(usize index, ref mut u8[] buffer) {
    let raw_process_args_result result = unsafe {
        process_arg_raw(index, buffer as ptr mut u8, buffer.len)
    };
    return checked_arg_result(result.status, result.needed);
}
