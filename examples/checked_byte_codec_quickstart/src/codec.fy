module codec;

import std::num as num;

pub enum EncodeError { Negative; InvalidWidth; Capacity; HighBits; Conversion; }

// This example owns unsigned little-endian field policy. Conversion owns only a byte.
// Complete preflight before the first write; failures preserve the caller's buffer.
pub fn Result<usize, EncodeError> encode_le(i64 input, usize width, ref mut u8[] output) {
    if (input < 0) { return .err(.Negative); }
    if (width == 0 || width > 8) { return .err(.InvalidWidth); }
    if (width > output.len) { return .err(.Capacity); }
    var i64 excess = input;
    var usize at = 0;
    while (at < width) { excess /= 256; at += 1; }
    if (excess != 0) { return .err(.HighBits); }
    var i64 remaining = input;
    at = 0;
    while (at < width) {
        // Every digit is 0..255 after the negative-input check.
        let u8 digit = switch (num::u8_from_i64(remaining % 256)) {
            case .ok(value): { value }
            case .err(_): { return .err(.Conversion); }
        };
        output[at] = digit;
        remaining /= 256;
        at += 1;
    }
    return .ok(width);
}
