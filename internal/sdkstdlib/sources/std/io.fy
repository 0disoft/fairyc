module std::io;

@repr(C)
struct raw_console_write_result {
    u32 status;
    usize written;
}

/// Reports the raw failure status and the byte prefix completed before failure.
pub struct ConsoleError {
    u32 status;
    usize written;
}

@effects(console)
@link_name("fy_std_stdout_write_all_v0")
unsafe extern C fn raw_console_write_result stdout_write_all_raw(ptr u8 data, usize length) ;

@effects(console)
@link_name("fy_std_stderr_write_all_v0")
unsafe extern C fn raw_console_write_result stderr_write_all_raw(ptr u8 data, usize length) ;

fn Result<usize, ConsoleError> checked_write_result(u32 status, usize written) {
    return if (status == 0) {
        .ok(written)
    } else {
        .err(ConsoleError {
            .status = status,
            .written = written,
        })
    };
}

/// Writes the exact UTF-8 bytes in text to stdout without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write(ref str text) {
    let raw_console_write_result result = unsafe {
        stdout_write_all_raw(text as ptr u8, text.len)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes the exact bytes in data to stdout without UTF-8 validation or a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_bytes(ref u8[] data) {
    let raw_console_write_result result = unsafe {
        stdout_write_all_raw(data as ptr u8, data.len)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes the exact UTF-8 bytes in text to stderr without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_stderr(ref str text) {
    let raw_console_write_result result = unsafe {
        stderr_write_all_raw(text as ptr u8, text.len)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes the exact bytes in data to stderr without UTF-8 validation or a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_stderr_bytes(ref u8[] data) {
    let raw_console_write_result result = unsafe {
        stderr_write_all_raw(data as ptr u8, data.len)
    };
    return checked_write_result(result.status, result.written);
}
