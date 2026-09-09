module numbers;
import std::bytes as bytes;

fn bool digit(u8 c) { return c >= 48 && c <= 57; }
fn bool identifier(u8 c) { return digit(c) || c == 95 || (c >= 65 && c <= 90) || (c >= 97 && c <= 122); }
fn bool put(ref mut bytes::Buffer output, u8 c) {
    return switch (bytes::append_byte(output, c)) { case .ok(_) : { true } case .err(_) : { false } };
}
fn u8 value(u8 c) {
    if (digit(c)) { return c - 48; }
    if (c >= 65 && c <= 70) { return c - 65 + 10; }
    if (c >= 97 && c <= 102) { return c - 97 + 10; }
    return 255;
}

fn i32 integer(ref u8[] raw, ref mut bytes::Buffer output) {
    var u8 base = 10;
    var usize first = 0;
    if (raw.len >= 2 && raw[0] == 48) {
        if (raw[1] == 120) { base = 16; first = 2; }
        else if (raw[1] == 98) { base = 2; first = 2; }
    }
    if (first == raw.len) { return 1; }
    var usize i = first;
    while (i < raw.len) {
        if (raw[i] == 95) {
            if (i == first || i + 1 == raw.len || value(raw[i-1]) >= base || value(raw[i+1]) >= base) { return 1; }
        } else if (value(raw[i]) >= base) { return 1; }
        i += 1;
    }
    // Little-endian decimal digits avoid restricting literals to machine integers.
    var bytes::Buffer digits = bytes::empty();
    if (!put(ref mut digits, 0)) { return 3; }
    i = first;
    while (i < raw.len) {
        if (raw[i] != 95) {
            var u8 carry = value(raw[i]);
            {
                let ref mut u8[] data = bytes::view_mut(ref mut digits);
                var usize j = 0;
                while (j < data.len) {
                    let u8 n = data[j] * base + carry;
                    data[j] = n % 10; carry = n / 10; j += 1;
                }
            }
            while (carry > 0) {
                if (!put(ref mut digits, carry % 10)) { return 3; }
                carry /= 10;
            }
        }
        i += 1;
    }
    let ref u8[] data = bytes::view(ref digits);
    i = data.len;
    while (i > 0) { i -= 1; if (!put(output, data[i] + 48)) { return 3; } }
    return 0;
}

fn i32 decimal(ref u8[] raw, ref mut bytes::Buffer output) {
    var bytes::Buffer clean = bytes::empty();
    var usize i = 0;
    while (i < raw.len) { if (raw[i] != 95 && !put(ref mut clean, raw[i])) { return 3; } i += 1; }
    let ref u8[] text = bytes::view(ref clean);
    var usize mantissa_end = text.len;
    i = 0;
    while (i < text.len) { if (text[i] == 101 || text[i] == 69) { mantissa_end = i; break; } i += 1; }
    var i32 exponent = 0;
    var bool negative = false;
    var bool too_large = false;
    if (mantissa_end < text.len) {
        i = mantissa_end + 1;
        if (i < text.len && (text[i] == 43 || text[i] == 45)) { negative = text[i] == 45; i += 1; }
        if (i == text.len) { return 1; }
        while (i < text.len) {
            if (!digit(text[i])) { return 1; }
            if (!too_large) {
                exponent = exponent * 10 + (text[i] - 48) as i32;
                if (exponent > 16384) { too_large = true; }
            }
            i += 1;
        }
    }
    if (mantissa_end == 0) { return 1; }
    var bytes::Buffer significant = bytes::empty();
    var bool dot = false;
    var i32 fraction = 0;
    i = 0;
    while (i < mantissa_end) {
        if (text[i] == 46) {
            if (dot || i == 0 || i + 1 == mantissa_end) { return 1; }
            dot = true;
        } else {
            if (!digit(text[i])) { return 1; }
            if (!put(ref mut significant, text[i])) { return 3; }
            if (dot) { fraction += 1; }
        }
        i += 1;
    }
    if (too_large) { return 4; }
    if (negative) { exponent = -exponent; }
    let ref u8[] data = bytes::view(ref significant);
    var usize first = 0;
    while (first + 1 < data.len && data[first] == 48) { first += 1; }
    var i32 width = 0;
    i = first;
    while (i < data.len) { width += 1; i += 1; }
    let i32 places = fraction - exponent;
    let i32 length = if (places <= 0) { width - places } else if (places >= width) { places + 2 } else { width + 1 };
    if (length > 16384) { return 4; }
    if (places >= width) {
        if (!put(output, 48) || !put(output, 46)) { return 3; }
        var i32 zeros = places - width;
        while (zeros > 0) { if (!put(output, 48)) { return 3; } zeros -= 1; }
    }
    i = first;
    var i32 offset = 0;
    while (i < data.len) {
        if (places > 0 && places < width && offset == width - places) { if (!put(output, 46)) { return 3; } }
        if (!put(output, data[i])) { return 3; }
        i += 1;
        offset += 1;
    }
    if (places <= 0) {
        var i32 zeros = -places;
        while (zeros > 0) { if (!put(output, 48)) { return 3; } zeros -= 1; }
    } else {
        var usize retained = 0;
        {
            let ref mut u8[] data_out = bytes::view_mut(output);
            retained = data_out.len;
            while (retained > 0 && data_out[retained-1] == 48) { retained -= 1; }
            if (retained > 0 && data_out[retained-1] == 46) { retained -= 1; }
        }
        if (!switch (bytes::truncate(output, retained)) { case .ok(_) : { true } case .err(_) : { false } }) { return 3; }
    }
    return 0;
}

// Status 1 is a complete malformed numeric token (E0004), not a scanner failure.
pub struct Scan { usize end; u32 kind; i32 status; }

pub fn Scan scan(ref u8[] input, usize start, ref mut bytes::Buffer output) {
    var usize p = start + 1;
    var u32 kind = 5;
    if (p < input.len && input[start] == 48 && (input[p] == 120 || input[p] == 98)) {
        p += 1;
        while (p < input.len && identifier(input[p])) { p += 1; }
    } else {
        while (p < input.len && (digit(input[p]) || input[p] == 95)) { p += 1; }
        if (p + 1 < input.len && input[p] == 46 && digit(input[p+1])) {
            kind = 6; p += 2;
            while (p < input.len && (digit(input[p]) || input[p] == 95)) { p += 1; }
        }
        if (p < input.len && (input[p] == 101 || input[p] == 69)) {
            kind = 6; p += 1;
            if (p < input.len && (input[p] == 43 || input[p] == 45)) { p += 1; }
            while (p < input.len && (digit(input[p]) || input[p] == 95)) { p += 1; }
        }
        while (p < input.len && identifier(input[p])) { p += 1; }
    }
    let i32 status = if (kind == 5) { integer(input[start..p], output) } else { decimal(input[start..p], output) };
    return Scan { .end = p, .kind = kind, .status = status };
}
