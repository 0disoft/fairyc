module strings;
import std::bytes as bytes;
import std::vec as v;
import diagnostics as diagnostics;
import utf8 as utf8;

pub struct Step { usize end; i32 status; usize diagnostics; }

fn bool put(ref mut bytes::Buffer output, u8 value) {
    return switch (bytes::append_byte(output, value)) { case .ok(_) : { true } case .err(_) : { false } };
}
fn u32 hex(u8 value) {
    if (value >= 48 && value <= 57) { return (value - 48) as u32; }
    if (value >= 65 && value <= 70) { return (value - 65) as u32 + 10; }
    if (value >= 97 && value <= 102) { return (value - 97) as u32 + 10; }
    return 255;
}
fn u8 octet(u32 value) {
    assert(value <= 255);
    let ref u8[] nibble = b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";
    return nibble[(value / 16) as usize] * 16 + nibble[(value % 16) as usize];
}
fn Step error(ref mut v::Vec<diagnostics::Entry> errors, usize count, usize limit, u32 code, usize start, usize end) {
    let i32 status = diagnostics::add_limited(errors, count, limit, code, start, end, false, b"");
    return Step { .end = end, .status = status, .diagnostics = count + if (status == 0) { 1 } else { 0 } };
}
fn bool encode(ref mut bytes::Buffer output, u32 value) {
    if (value < 128) { return put(output, octet(value)); }
    if (value < 2048) {
        return put(output, octet(value / 64 + 192)) && put(output, octet(value % 64 + 128));
    }
    if (value < 65536) {
        return put(output, octet(value / 4096 + 224)) && put(output, octet((value / 64) % 64 + 128)) && put(output, octet(value % 64 + 128));
    }
    return put(output, octet(value / 262144 + 240)) && put(output, octet((value / 4096) % 64 + 128)) && put(output, octet((value / 64) % 64 + 128)) && put(output, octet(value % 64 + 128));
}

fn Step escape(ref u8[] input, usize start, u32 kind, ref mut bytes::Buffer output, ref mut v::Vec<diagnostics::Entry> errors, usize count, usize limit) {
    var usize p = start + 1;
    if (p == input.len) { return error(errors, count, limit, 5, start, p); }
    let u8 current = input[p]; p += 1;
    var u8 value = current;
    if (current == 92 || current == 34 || current == 110 || current == 114 || current == 116 || current == 48) {
        if (current == 110) { value = 10; }
        else if (current == 114) { value = 13; }
        else if (current == 116) { value = 9; }
        else if (current == 48) { value = 0; }
        if (!put(output, value)) { return Step { .end = p, .status = 3, .diagnostics = count }; }
        if (current == 48 && kind == 9) { return error(errors, count, limit, 6, start, p); }
    } else if (current == 120) {
        if (input.len - p < 2 || hex(input[p]) >= 16 || hex(input[p+1]) >= 16) {
            while (p < input.len && p < start + 4 && hex(input[p]) < 16) { p += 1; }
            return error(errors, count, limit, 5, start, p);
        }
        value = octet(hex(input[p]) * 16 + hex(input[p+1])); p += 2;
        if (!put(output, value)) { return Step { .end = p, .status = 3, .diagnostics = count }; }
        if (kind == 9 && value == 0) { return error(errors, count, limit, 6, start, p); }
    } else if (current == 117) {
        if (kind != 7 || p == input.len || input[p] != 123) { return error(errors, count, limit, 5, start, p); }
        p += 1;
        let usize digits_start = p;
        var u32 scalar = 0;
        while (p < input.len && hex(input[p]) < 16 && p - digits_start < 6) { scalar = scalar * 16 + hex(input[p]); p += 1; }
        if (p == digits_start || p == input.len || input[p] != 125) {
            while (p < input.len && input[p] != 125 && input[p] != 34) { p += 1; }
            if (p < input.len && input[p] == 125) { p += 1; }
            return error(errors, count, limit, 5, start, p);
        }
        p += 1;
        if (scalar > 1114111 || (scalar >= 55296 && scalar <= 57343)) { return error(errors, count, limit, 5, start, p); }
        if (!encode(output, scalar)) { return Step { .end = p, .status = 3, .diagnostics = count }; }
    } else { return error(errors, count, limit, 5, start, p); }
    return Step { .end = p, .status = 0, .diagnostics = count };
}

pub fn Step scan(ref u8[] input, usize start, u32 kind, usize opener, ref mut bytes::Buffer output, ref mut v::Vec<diagnostics::Entry> errors, usize count, usize limit) {
    var usize p = start + opener;
    var usize diagnostic_count = count;
    var bytes::Buffer raw = bytes::empty();
    var bool terminated = false;
    while (p < input.len) {
        let u8 current = input[p];
        if (current == 34) { p += 1; terminated = true; break; }
        if (current == 10 || current == 13) {
            let i32 status = diagnostics::add_limited(errors, diagnostic_count, limit, 5, p, p+1, true, b"\"");
            if (status != 0) { return Step { .end = p, .status = status, .diagnostics = diagnostic_count }; }
            diagnostic_count += 1;
            break;
        }
        if (current == 92) {
            let Step step = escape(input, p, kind, ref mut raw, errors, diagnostic_count, limit);
            if (step.status != 0) { return step; }
            diagnostic_count = step.diagnostics;
            p = step.end; continue;
        }
        if (current == 0) {
            let u32 code = if (kind == 9) { 6 } else { 5 };
            let i32 status = diagnostics::add_limited(errors, diagnostic_count, limit, code, p, p+1, false, b"");
            if (status != 0) { return Step { .end = p+1, .status = status, .diagnostics = diagnostic_count }; }
            diagnostic_count += 1;
        }
        let usize width = utf8::width(input, p);
        if (width == 0) {
            let i32 status = diagnostics::add_limited(errors, diagnostic_count, limit, 1, p, p+1, false, b"");
            if (status != 0) { return Step { .end = p+1, .status = status, .diagnostics = diagnostic_count }; }
            diagnostic_count += 1;
            p += 1; continue;
        }
        var usize i = 0;
        while (i < width) { if (!put(ref mut raw, input[p+i])) { return Step { .end = p, .status = 3, .diagnostics = diagnostic_count }; } i += 1; }
        p += width;
    }
    if (!terminated && p == input.len) {
        let i32 status = diagnostics::add_limited(errors, diagnostic_count, limit, 5, start, p, true, b"\"");
        if (status != 0) { return Step { .end = p, .status = status, .diagnostics = diagnostic_count }; }
        diagnostic_count += 1;
    }
    if (kind == 7 && terminated) {
        let ref u8[] data = bytes::view(ref raw);
        var usize i = 0;
        while (i < data.len) {
            let usize width = utf8::width(data, i);
            if (width == 0) {
                let i32 status = diagnostics::add_limited(errors, diagnostic_count, limit, 5, start, p, false, b"");
                if (status != 0) { return Step { .end = p, .status = status, .diagnostics = diagnostic_count }; }
                diagnostic_count += 1;
                break;
            }
            i += width;
        }
    }
    if (kind == 9 && !put(ref mut raw, 0)) { return Step { .end = p, .status = 3, .diagnostics = diagnostic_count }; }
    let ref u8[] data = bytes::view(ref raw);
    let ref u8[] alphabet = b"0123456789abcdef";
    var usize i = 0;
    while (i < data.len) {
        if (!put(output, alphabet[(data[i] / 16) as usize]) || !put(output, alphabet[(data[i] % 16) as usize])) { return Step { .end = p, .status = 3, .diagnostics = diagnostic_count }; }
        i += 1;
    }
    return Step { .end = p, .status = 0, .diagnostics = diagnostic_count };
}
