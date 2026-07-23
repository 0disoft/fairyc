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

/// Writes the exact UTF-8 bytes in text to stdout without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write(ref str text) {
    let raw_console_write_result result = unsafe {
        stdout_write_all_raw(text as ptr u8, text.len)
    };
    return if (result.status == 0) {
        .ok(result.written)
    } else {
        .err(ConsoleError {
            .status = result.status,
            .written = result.written,
        })
    };
}
