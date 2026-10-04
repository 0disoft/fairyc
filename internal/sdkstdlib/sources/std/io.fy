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
@link_name("fy_std_stderr_write_i32_v0")
unsafe extern C fn raw_console_write_result stderr_write_i32_raw(i32 value) ;

@effects(console)
@link_name("fy_std_stderr_write_u32_v0")
unsafe extern C fn raw_console_write_result stderr_write_u32_raw(u32 value) ;

@effects(console)
@link_name("fy_std_stderr_write_usize_v0")
unsafe extern C fn raw_console_write_result stderr_write_usize_raw(usize value) ;

@effects(console)
@link_name("fy_std_stdin_read_v0")
unsafe extern C fn raw_console_read_result stdin_read_raw(ptr mut u8 data, usize length) ;

fn u8 signed_decimal_digit(i64 value) {
    return if (value == -9 || value == 9) {
        57
    } else if (value == -8 || value == 8) {
        56
    } else if (value == -7 || value == 7) {
        55
    } else if (value == -6 || value == 6) {
        54
    } else if (value == -5 || value == 5) {
        53
    } else if (value == -4 || value == 4) {
        52
    } else if (value == -3 || value == 3) {
        51
    } else if (value == -2 || value == 2) {
        50
    } else if (value == -1 || value == 1) {
        49
    } else {
        48
    };
}

// Fills a caller-owned 20-byte scratch view and returns its encoded suffix start.
// Keep the signed remainder: never negate the i64 minimum value.
fn usize encode_i64_decimal(ref mut u8[] digits, i64 value) {
    var usize position = 20;
    var i64 remaining = value;
    while (remaining <= -10 || remaining >= 10) {
        position -= 1;
        digits[position] = signed_decimal_digit(remaining % 10);
        remaining /= 10;
    }
    position -= 1;
    digits[position] = signed_decimal_digit(remaining);
    if (value < 0) {
        position -= 1;
        digits[position] = 45;
    }
    return position;
}

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

/// Writes value as locale-independent decimal ASCII without allocation or a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_i64(i64 value) {
    var u8[20] digits = [0; 20];
    let usize position = encode_i64_decimal(ref mut digits, value);
    let ref u8[] encoded = ref digits;
    return write_bytes(encoded[position..]);
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

/// Writes value as locale-independent decimal ASCII to stderr without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_stderr_i32(i32 value) {
    let raw_console_write_result result = unsafe {
        stderr_write_i32_raw(value)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes value as locale-independent decimal ASCII without allocation or a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_stderr_i64(i64 value) {
    var u8[20] digits = [0; 20];
    let usize position = encode_i64_decimal(ref mut digits, value);
    let ref u8[] encoded = ref digits;
    return write_stderr_bytes(encoded[position..]);
}

/// Writes value as locale-independent decimal ASCII to stderr without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_stderr_u32(u32 value) {
    let raw_console_write_result result = unsafe {
        stderr_write_u32_raw(value)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes value as locale-independent decimal ASCII to stderr without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_stderr_usize(usize value) {
    let raw_console_write_result result = unsafe {
        stderr_write_usize_raw(value)
    };
    return checked_write_result(result.status, result.written);
}

/// Writes the exact ASCII text true or false to stderr without adding a newline.
@effects(console)
pub fn Result<usize, ConsoleError> write_stderr_bool(bool value) {
    return if (value) {
        write_stderr("true")
    } else {
        write_stderr("false")
    };
}

/// Reads at most buffer.len exact bytes from stdin; a successful zero count is end-of-file.
@effects(console)
pub fn Result<usize, ConsoleReadError> read_stdin(ref mut u8[] buffer) {
    let raw_console_read_result result = unsafe {
        stdin_read_raw(buffer as ptr mut u8, buffer.len)
    };
    return checked_read_result(result.status, result.read);
}
