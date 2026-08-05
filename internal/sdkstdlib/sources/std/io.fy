module std::io;

@repr(C)
struct raw_console_write_result {
    u32 status;
    usize written;
}

@repr(C)
struct raw_console_read_result {
    u32 status;
    usize read;
}

/// Reports the raw failure status and the byte prefix completed before failure.
pub struct ConsoleError {
    u32 status;
    usize written;
}

/// Reports the raw failure status without treating an ordinary end-of-file as an error.
pub struct ConsoleReadError {
    u32 status;
    usize read;
}

@effects(console)
@link_name("fy_std_stdout_write_all_v0")
unsafe extern C fn raw_console_write_result stdout_write_all_raw(ptr u8 data, usize length) ;

@effects(console)
@link_name("fy_std_stdout_write_i32_v0")
unsafe extern C fn raw_console_write_result stdout_write_i32_raw(i32 value) ;

@effects(console)
@link_name("fy_std_stdout_write_u32_v0")
unsafe extern C fn raw_console_write_result stdout_write_u32_raw(u32 value) ;

@effects(console)
@link_name("fy_std_stdout_write_usize_v0")
unsafe extern C fn raw_console_write_result stdout_write_usize_raw(usize value) ;

@effects(console)
@link_name("fy_std_stderr_write_all_v0")
unsafe extern C fn raw_console_write_result stderr_write_all_raw(ptr u8 data, usize length) ;

@effects(console)
@link_name("fy_std_stdin_read_v0")
unsafe extern C fn raw_console_read_result stdin_read_raw(ptr mut u8 data, usize length) ;

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

fn Result<usize, ConsoleReadError> checked_read_result(u32 status, usize read) {
    return if (status == 0) {
        .ok(read)
    } else {
        .err(ConsoleReadError {
            .status = status,
            .read = read,
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

/// Writes value as locale-independent decimal ASCII to stdout without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_i32(i32 value) {
    let raw_console_write_result result = unsafe {
        stdout_write_i32_raw(value)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes value as locale-independent decimal ASCII to stdout without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_u32(u32 value) {
    let raw_console_write_result result = unsafe {
        stdout_write_u32_raw(value)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes value as locale-independent decimal ASCII to stdout without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_usize(usize value) {
    let raw_console_write_result result = unsafe {
        stdout_write_usize_raw(value)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes the exact ASCII text true or false to stdout without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_bool(bool value) {
    return if (value) {
        write("true")
    } else {
        write("false")
    };
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

/// Reads at most buffer.len exact bytes from stdin; a successful zero count is end-of-file.
@effects(console)
pub fn Result<usize, ConsoleReadError> read_stdin(ref mut u8[] buffer) {
    let raw_console_read_result result = unsafe {
        stdin_read_raw(buffer as ptr mut u8, buffer.len)
    };
    return checked_read_result(result.status, result.read);
}
