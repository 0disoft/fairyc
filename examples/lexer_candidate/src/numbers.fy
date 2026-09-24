module numbers;
import std::bytes as bytes;
import budget as budget;

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
    var bool plain_decimal = base == 10 && (raw.len == 1 || raw[0] != 48);
    var usize i = first;
    while (i < raw.len) {
        if (raw[i] == 95) {
            plain_decimal = false;
            if (i == first || i + 1 == raw.len || value(raw[i-1]) >= base || value(raw[i+1]) >= base) { return 1; }
        } else if (value(raw[i]) >= base) { return 1; }
        i += 1;
    }
    if (plain_decimal) {
        return switch (bytes::append(output, raw)) { case .ok(_): { 0 } case .err(_): { 3 } };
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

struct Cook { i32 status; budget::Work work; }

fn Cook decimal(ref u8[] raw, ref mut bytes::Buffer output, budget::Work initial) {
    var budget::Work work = initial;
    var bytes::Buffer clean = bytes::empty();
    var usize i = 0;
    while (i < raw.len) { if (raw[i] != 95 && !put(ref mut clean, raw[i])) { return Cook { .status = 3, .work = work }; } i += 1; }
    let ref u8[] text = bytes::view(ref clean);
    var usize mantissa_end = text.len;
    i = 0;
    while (i < text.len) { if (text[i] == 101 || text[i] == 69) { mantissa_end = i; break; } i += 1; }
    let usize max_int = 9223372036854775807;
    var usize exponent = 0;
    var bool negative = false;
    var bool overflow = false;
    if (mantissa_end < text.len) {
        i = mantissa_end + 1;
        if (i < text.len && (text[i] == 43 || text[i] == 45)) { negative = text[i] == 45; i += 1; }
        if (i == text.len) { return Cook { .status = 1, .work = work }; }
        let usize maximum = if (negative) { max_int + 1 } else { max_int };
        while (i < text.len) {
            if (!digit(text[i])) { return Cook { .status = 1, .work = work }; }
            let usize next = (text[i] - 48) as usize;
            if (!overflow) {
                if (exponent > maximum / 10 || (exponent == maximum / 10 && next > maximum % 10)) { overflow = true; }
                else { exponent = exponent * 10 + next; }
            }
            i += 1;
        }
    }
    if (overflow || mantissa_end == 0) { return Cook { .status = 1, .work = work }; }
    var bytes::Buffer significant = bytes::empty();
    var bool dot = false;
    var usize fraction = 0;
    i = 0;
    while (i < mantissa_end) {
        if (text[i] == 46) {
            if (dot || i == 0 || i + 1 == mantissa_end) { return Cook { .status = 1, .work = work }; }
            dot = true;
        } else {
            if (!digit(text[i])) { return Cook { .status = 1, .work = work }; }
            if (!put(ref mut significant, text[i])) { return Cook { .status = 3, .work = work }; }
            if (dot) { fraction += 1; }
        }
        i += 1;
    }
    let ref u8[] data = bytes::view(ref significant);
    var usize first = 0;
    while (first + 1 < data.len && data[first] == 48) { first += 1; }
    let usize width = data.len - first;
    let bool trailing = (!negative || exponent == 0) && exponent >= fraction;
    var usize places = 0;
    var usize zeros = 0;
    var usize length = 0;
    var bool unrepresentable = false;
    if (trailing) {
        zeros = exponent - fraction;
        if (width > max_int - zeros) { unrepresentable = true; }
        else { length = width + zeros; }
    } else {
        if (negative) {
            if (exponent > max_int - fraction) { unrepresentable = true; }
            else { places = fraction + exponent; }
        } else { places = fraction - exponent; }
        if (!unrepresentable) {
            if (places >= width) {
                if (places > max_int - 2) { unrepresentable = true; }
                else { length = places + 2; }
            } else { length = width + 1; }
        }
    }
    if (unrepresentable) {
        return Cook { .status = 5, .work = work };
    }
    // Charge the full expansion before materialization, including later-trimmed zeros.
    if (!budget::charge(ref mut work, length)) { return Cook { .status = 5, .work = work }; }
    if (length > 16384) { return Cook { .status = 4, .work = work }; }
    if (!trailing && places >= width) {
        if (!put(output, 48) || !put(output, 46)) { return Cook { .status = 3, .work = work }; }
        var usize leading = places - width;
        while (leading > 0) { if (!put(output, 48)) { return Cook { .status = 3, .work = work }; } leading -= 1; }
    }
    i = first;
    var usize offset = 0;
    while (i < data.len) {
        if (!trailing && places < width && offset == width - places) { if (!put(output, 46)) { return Cook { .status = 3, .work = work }; } }
        if (!put(output, data[i])) { return Cook { .status = 3, .work = work }; }
        i += 1; offset += 1;
    }
    if (trailing) {
        while (zeros > 0) { if (!put(output, 48)) { return Cook { .status = 3, .work = work }; } zeros -= 1; }
    } else {
        var usize retained = 0;
        {
            let ref mut u8[] data_out = bytes::view_mut(output);
            retained = data_out.len;
            while (retained > 0 && data_out[retained-1] == 48) { retained -= 1; }
            if (retained > 0 && data_out[retained-1] == 46) { retained -= 1; }
        }
        if (!switch (bytes::truncate(output, retained)) { case .ok(_) : { true } case .err(_) : { false } }) { return Cook { .status = 3, .work = work }; }
    }
    return Cook { .status = 0, .work = work };
}

// Status 1 is a complete malformed numeric token; 5 is work exhaustion.
pub struct Scan { usize end; u32 kind; i32 status; budget::Work work; }

pub fn Scan scan(ref u8[] input, usize start, ref mut bytes::Buffer output, budget::Work initial) {
    var budget::Work work = initial;
    var usize p = start + 1;
    var u32 kind = 5;
    if (p < input.len && input[start] == 48 && (input[p] == 120 || input[p] == 98)) {
        if (!budget::charge(ref mut work, 1)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
        p += 1;
        while (p < input.len && identifier(input[p])) {
            if (!budget::charge(ref mut work, 1)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
            p += 1;
        }
    } else {
        while (p < input.len && (digit(input[p]) || input[p] == 95)) {
            if (!budget::charge(ref mut work, 1)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
            p += 1;
        }
        if (p + 1 < input.len && input[p] == 46 && digit(input[p+1])) {
            kind = 6;
            if (!budget::charge(ref mut work, 2)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
            p += 2;
            while (p < input.len && (digit(input[p]) || input[p] == 95)) {
                if (!budget::charge(ref mut work, 1)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
                p += 1;
            }
        }
        if (p < input.len && (input[p] == 101 || input[p] == 69)) {
            kind = 6;
            if (!budget::charge(ref mut work, 1)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
            p += 1;
            if (p < input.len && (input[p] == 43 || input[p] == 45)) {
                if (!budget::charge(ref mut work, 1)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
                p += 1;
            }
            while (p < input.len && (digit(input[p]) || input[p] == 95)) {
                if (!budget::charge(ref mut work, 1)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
                p += 1;
            }
        }
        while (p < input.len && identifier(input[p])) {
            if (!budget::charge(ref mut work, 1)) { return Scan { .end = p, .kind = kind, .status = 5, .work = work }; }
            p += 1;
        }
    }
    if (kind == 5) {
        let i32 status = integer(input[start..p], output);
        return Scan { .end = p, .kind = kind, .status = status, .work = work };
    }
    let Cook result = decimal(input[start..p], output, work);
    return Scan { .end = p, .kind = kind, .status = result.status, .work = result.work };
}
