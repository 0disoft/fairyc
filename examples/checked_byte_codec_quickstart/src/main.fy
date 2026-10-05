module main;

import codec as codec;
import numbers as numbers;
import std::io as io;
import std::num as num;
import std::process as process;

fn bool equals(ref u8[] left, ref u8[] right) {
    if (left.len != right.len) { return false; }
    var usize at = 0;
    while (at < left.len) { if (left[at] != right[at]) { return false; } at += 1; }
    return true;
}

@effects(console)
fn bool stderr(ref str message) {
    return switch (io::write_stderr(message)) {
        case .ok(count): { count == message.len }
        case .err(_): { false }
    };
}

@effects(console)
fn i32 error(ref str message, i32 code) {
    if (!stderr("checked-byte-codec: ") || !stderr(message) || !stderr("\n")) { return 3; }
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
    };
    if (!stderr("checked-byte-codec: ") || !stderr(name) || !stderr(": ") || !stderr(message) || !stderr("\n")) { return 3; }
    return 1;
}

@effects(console)
fn i32 render(ref codec::Packet packet) {
    let ref u8[] alphabet = b"0123456789abcdef";
    var u8[24] output = [0; 24];
    var usize at = 0;
    for (let u8 byte in packet.bytes[..packet.length]) {
        let usize high = (byte / 16) as usize;
        let usize low = (byte % 16) as usize;
        output[at * 3] = alphabet[high];
        output[at * 3 + 1] = alphabet[low];
        output[at * 3 + 2] = if (at + 1 == packet.length) { 10 } else { 32 };
        at += 1;
    }
    let ref u8[] view = ref output;
    return switch (io::write_bytes(view[..packet.length * 3])) {
        case .ok(count): { if (count == packet.length * 3) { 0 } else { 3 } }
        case .err(_): { 3 }
    };
}

@effects(console)
fn i32 render_decimal(i64 value) {
    var i64 remaining = value;
    var usize length = 1;
    while (remaining >= 10) { remaining /= 10; length += 1; }
    if (!switch (io::write_i64(value)) {
        case .ok(count): { count == length }
        case .err(_): { false }
    }) { return 3; }
    return switch (io::write("\n")) {
        case .ok(count): { if (count == 1) { 0 } else { 3 } }
        case .err(_): { 3 }
    };
}

@effects(console)
fn i32 decode_argument() {
    var u8[16] raw = [0; 16];
    let usize used = switch (process::arg(2, ref mut raw)) {
        case .ok(count): { count }
        case .err(failure): {
            return error(if (failure.status == 3) { "HEX: expected at most 16 hex digits" } else { "HEX: cannot read argument" }, 1);
        }
    };
    let ref u8[] view = ref raw;
    let i64 value = switch (codec::decode_hex_le(view[..used])) {
        case .ok(decoded): { decoded }
        case .err(failure): {
            let ref str message = switch (failure) {
                case .Empty: { "HEX: empty input" }
                case .OddLength: { "HEX: expected pairs of hex digits" }
                case .TooLong: { "HEX: expected at most 16 hex digits" }
                case .InvalidHex: { "HEX: expected ASCII hex digits without separators" }
                case .OutsideI64: { "HEX: value is outside nonnegative i64 range" }
            };
            return error(message, 1);
        }
    };
    return render_decimal(value);
}

@effects(console)
fn i32 main() {
    let usize argc = process::arg_count();
    if (argc < 3 || argc > 4) { return error("usage: checked-byte-codec byte VALUE | le VALUE WIDTH | decode-le HEX", 2); }
    var u8[9] mode = [0; 9];
    let usize used = switch (process::arg(1, ref mut mode)) {
        case .ok(count): { count }
        case .err(_): { return error("usage: checked-byte-codec byte VALUE | le VALUE WIDTH | decode-le HEX", 2); }
    };
    let ref u8[] mode_view = ref mode;
    let bool single = equals(mode_view[..used], b"byte");
    let bool little = equals(mode_view[..used], b"le");
    let bool decoding = equals(mode_view[..used], b"decode-le");
    if ((!single && !little && !decoding) || ((single || decoding) && argc != 3) || (little && argc != 4)) {
        return error("usage: checked-byte-codec byte VALUE | le VALUE WIDTH | decode-le HEX", 2);
    }
    if (decoding) { return decode_argument(); }
    let i64 value = switch (numbers::argument(2)) {
        case .ok(parsed): { parsed }
        case .err(failure): { return input_error("VALUE", failure); }
    };
    var codec::Packet packet = codec::Packet { .bytes = [0; 8], .length = 0 };
    var usize width = 1;
    if (single) {
        packet.bytes[0] = switch (num::u8_from_i64(value)) {
            case .ok(byte): { byte }
            case .err(failure): {
                let ref str message = switch (failure) {
                    case .Negative: { "VALUE: negative value" }
                    case .OutOfRange: { "VALUE: outside byte range 0..255" }
                };
                return error(message, 1);
            }
        };
        packet.length = 1;
    } else {
        let i64 requested = switch (numbers::argument(3)) {
            case .ok(parsed): { parsed }
            case .err(failure): { return input_error("WIDTH", failure); }
        };
        if (requested < 1 || requested > 8) { return error("WIDTH: expected 1..8", 1); }
        width = switch (num::usize_from_i64(requested)) {
            case .ok(converted): { converted }
            case .err(_): { return error("WIDTH: cannot convert width", 3); }
        };
        let usize written = switch (codec::encode_le(value, width, ref mut packet.bytes[..width])) {
            case .ok(count): { count }
            case .err(failure): {
                let ref str message = switch (failure) {
                    case .Negative: { "VALUE: negative value" }
                    case .InvalidWidth: { "WIDTH: expected 1..8" }
                    case .Capacity: { "output capacity is too small" }
                    case .HighBits: { "VALUE: does not fit WIDTH bytes" }
                    case .Conversion: { "cannot convert byte digit" }
                };
                let i32 code = switch (failure) {
                    case .Negative: { 1 } case .InvalidWidth: { 1 } case .HighBits: { 1 }
                    case .Capacity: { 3 } case .Conversion: { 3 }
                };
                return error(message, code);
            }
        };
        if (written != width) { return error("incorrect encoded length", 3); }
        packet.length = written;
    }
    return render(ref packet);
}
