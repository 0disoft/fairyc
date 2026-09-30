module diagnostics;

import std::io as io;
import std::bytes as bytes;

@effects(console)
fn bool stderr_text(ref str value) {
    return switch (io::write_stderr(value)) {
        case .ok(count) : { count == value.len }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool stderr_number(usize value) {
    return switch (io::write_stderr_usize(value)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
}

@effects(console)
pub fn i32 error(ref str path, usize record, usize line, ref str message) {
    let bool path_written = stderr_text(path);
    if (record != 0) {
        let bool label = stderr_text(":record=");
        let bool number = stderr_number(record);
    }
    if (line != 0) {
        let bool label = stderr_text(":line=");
        let bool number = stderr_number(line);
    }
    let bool prefix = stderr_text(": error: ");
    let bool body = stderr_text(message);
    let bool end = stderr_text("\n");
    return 1;
}

@effects(console)
pub fn i32 usage(ref str message) {
    let i32 reported = error("csv-aggregator", 0, 0, message);
    return 2;
}

@effects(console)
pub fn i32 file_error(ref str path, ref str operation, u32 status, u32 os_code) {
    let bool path_written = stderr_text(path);
    let bool prefix = stderr_text(": error: ");
    let bool body = stderr_text(operation);
    let bool status_label = stderr_text(" (status=");
    let bool status_written = switch (io::write_stderr_u32(status)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
    let bool os_label = stderr_text(" os_code=");
    let bool os_written = switch (io::write_stderr_u32(os_code)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
    let bool end = stderr_text(")\n");
    return 1;
}

@effects(console)
pub fn bool stdout_text(ref str message) {
    return switch (io::write(message)) {
        case .ok(count) : { count == message.len }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool stdout_signed(i64 value) {
    var bytes::Buffer buffer = bytes::empty();
    let bool formatted = switch (bytes::append_i64(ref mut buffer, value)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
    if (!formatted) { return false; }
    let ref u8[] encoded = bytes::view(ref buffer);
    return switch (io::write_bytes(encoded)) {
        case .ok(count) : { count == encoded.len }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool stdout_count(usize count) {
    return switch (io::write_usize(count)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
}

@effects(console)
pub fn i32 summary(usize count, i64 sum, i64 minimum, i64 maximum) {
    if (!stdout_text("count=") || !stdout_count(count) ||
        !stdout_text(" sum=") || !stdout_signed(sum) ||
        !stdout_text(" min=") || !stdout_signed(minimum) ||
        !stdout_text(" max=") || !stdout_signed(maximum) || !stdout_text("\n")) {
        return error("csv-aggregator", 0, 0, "stdout write failed");
    }
    return 0;
}
