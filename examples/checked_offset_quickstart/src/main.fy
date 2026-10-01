module main;

import numbers as numbers;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

@effects(console)
fn bool stderr(ref str message) {
    return switch (io::write_stderr(message)) {
        case .ok(count): { count == message.len }
        case .err(_): { false }
    };
}

@effects(console)
fn i32 error(ref str message, i32 code) {
    if (!stderr("checked-offset: ") || !stderr(message) || !stderr("\n")) { return 3; }
    return code;
}

@effects(console)
fn i32 input_error(ref str name, numbers::InputError failure) {
    let ref str message = switch (failure) {
        case .Unavailable: { "cannot read argument" }
        case .TooLong: { "argument exceeds 20 UTF-8 bytes" }
        case .Empty: { "empty integer" }
        case .InvalidDecimal: { "expected signed decimal integer" }
        case .OutsideI64: { "integer is outside i64 range" }
        case .Negative: { "negative value" }
        case .OutsideUsize: { "integer is outside usize range" }
    };
    if (!stderr("checked-offset: ") || !stderr(name) || !stderr(": ") || !stderr(message) || !stderr("\n")) { return 3; }
    return 1;
}

@effects(console)
fn Result<i32, bytes::BufferError> render(usize length, usize offset, usize count) {
    var bytes::Buffer output = bytes::with_capacity(128)?;
    let usize a = bytes::append(ref mut output, b"length=")?;
    let usize b = bytes::append_usize(ref mut output, length)?;
    let usize c = bytes::append(ref mut output, b" offset=")?;
    let usize d = bytes::append_usize(ref mut output, offset)?;
    let usize e = bytes::append(ref mut output, b" count=")?;
    let usize f = bytes::append_usize(ref mut output, count)?;
    let usize g = bytes::append(ref mut output, b" end=")?;
    let usize h = bytes::append_usize(ref mut output, offset + count)?;
    let usize i = bytes::append(ref mut output, b"\n")?;
    let ref u8[] encoded = bytes::view(ref output);
    return .ok(switch (io::write_bytes(encoded)) {
        case .ok(written): { if (written == encoded.len) { 0 } else { 3 } }
        case .err(_): { 3 }
    });
}

@effects(console)
fn i32 main() {
    if (process::arg_count() != 4) { return error("usage: checked-offset LENGTH OFFSET COUNT", 2); }
    let usize length = switch (numbers::argument(1)) {
        case .ok(value): { value } case .err(failure): { return input_error("LENGTH", failure); }
    };
    let usize offset = switch (numbers::argument(2)) {
        case .ok(value): { value } case .err(failure): { return input_error("OFFSET", failure); }
    };
    let usize count = switch (numbers::argument(3)) {
        case .ok(value): { value } case .err(failure): { return input_error("COUNT", failure); }
    };
    if (offset > length) { return error("offset exceeds length", 1); }
    // Only subtract after offset is known to fit; only add after count is known to fit.
    if (count > length - offset) { return error("count exceeds remaining length", 1); }
    return switch (render(length, offset, count)) {
        case .ok(code): { code }
        case .err(_): { error("cannot format output", 3) }
    };
}
