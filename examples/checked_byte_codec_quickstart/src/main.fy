module main;

import codec as codec;
import numbers as numbers;
import std::io as io;
import std::num as num;
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
fn i32 render(ref u8[] encoded) {
    let ref u8[] alphabet = b"0123456789abcdef";
    var u8[24] output = [0; 24];
    var usize at = 0;
    while (at < encoded.len) {
        let usize high = (encoded[at] / 16) as usize;
        let usize low = (encoded[at] % 16) as usize;
        output[at * 3] = alphabet[high];
        output[at * 3 + 1] = alphabet[low];
        output[at * 3 + 2] = if (at + 1 == encoded.len) { 10 } else { 32 };
        at += 1;
    }
    let ref u8[] view = ref output;
    return switch (io::write_bytes(view[..encoded.len * 3])) {
        case .ok(count): { if (count == encoded.len * 3) { 0 } else { 3 } }
        case .err(_): { 3 }
    };
}

@effects(console)
fn i32 main() {
    let usize argc = process::arg_count();
    if (argc < 3 || argc > 4) { return error("usage: checked-byte-codec byte VALUE | le VALUE WIDTH", 2); }
    var u8[4] mode = [0; 4];
    let usize used = switch (process::arg(1, ref mut mode)) {
        case .ok(count): { count }
        case .err(_): { return error("usage: checked-byte-codec byte VALUE | le VALUE WIDTH", 2); }
    };
    let bool single = used == 4 && mode[0] == 98 && mode[1] == 121 && mode[2] == 116 && mode[3] == 101;
    let bool little = used == 2 && mode[0] == 108 && mode[1] == 101;
    if ((!single && !little) || (single && argc != 3) || (little && argc != 4)) {
        return error("usage: checked-byte-codec byte VALUE | le VALUE WIDTH", 2);
    }
    let i64 value = switch (numbers::argument(2)) {
        case .ok(parsed): { parsed }
        case .err(failure): { return input_error("VALUE", failure); }
    };
    var u8[8] encoded = [0; 8];
    var usize width = 1;
    if (single) {
        encoded[0] = switch (num::u8_from_i64(value)) {
            case .ok(byte): { byte }
            case .err(failure): {
                let ref str message = switch (failure) {
                    case .Negative: { "VALUE: negative value" }
                    case .OutOfRange: { "VALUE: outside byte range 0..255" }
                };
                return error(message, 1);
            }
        };
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
        let usize written = switch (codec::encode_le(value, width, ref mut encoded)) {
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
    }
    let ref u8[] view = ref encoded;
    return render(view[..width]);
}
