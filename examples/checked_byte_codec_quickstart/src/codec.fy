module codec;

import std::num as num;

pub enum EncodeError { Negative; InvalidWidth; Capacity; HighBits; Conversion; }
pub enum DecodeError { Empty; OddLength; TooLong; InvalidHex; OutsideI64; }

pub struct Packet { u8[8] bytes; usize length; }

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

// Reconstruct from the most significant byte, guarding before multiplication.
pub fn Result<i64, DecodeError> decode_le(ref u8[] input) {
    if (input.len == 0) { return .err(.Empty); }
    if (input.len > 8) { return .err(.TooLong); }
    let i64 maximum = 9223372036854775807;
    var i64 value = 0;
    var usize at = input.len;
    while (at > 0) {
        at -= 1;
        let i64 digit = input[at] as i64;
        if (value > (maximum - digit) / 256) { return .err(.OutsideI64); }
        value = value * 256 + digit;
    }
    return .ok(value);
}

fn Result<u8, DecodeError> nibble(u8 digit) {
    if (digit >= 48 && digit <= 57) { return .ok(digit - 48); }
    if (digit >= 65 && digit <= 70) { return .ok(digit - 55); }
    if (digit >= 97 && digit <= 102) { return .ok(digit - 87); }
    return .err(.InvalidHex);
}

// Validate length and every ASCII hex digit before numeric range checking.
pub fn Result<i64, DecodeError> decode_hex_le(ref u8[] input) {
    if (input.len == 0) { return .err(.Empty); }
    if (input.len > 16) { return .err(.TooLong); }
    if (input.len % 2 != 0) { return .err(.OddLength); }
    let usize length = input.len / 2;
    var Packet packet = Packet { .bytes = [0; 8], .length = length };
    var usize at = 0;
    for (let ref mut u8 byte in packet.bytes[..length]) {
        let u8 high = nibble(input[at * 2])?;
        let u8 low = nibble(input[at * 2 + 1])?;
        *byte = high * 16 + low;
        at += 1;
    }
    return decode_le(ref packet.bytes[..packet.length]);
}
