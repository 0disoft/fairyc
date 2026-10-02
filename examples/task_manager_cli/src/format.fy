module format;
import std::bytes as bytes;
import std::text as text;
import output as output;

pub fn bool buffer_ok(Result<usize, bytes::BufferError> result) {
    return switch (result) { case .ok(_) : { true } case .err(_) : { false } };
}
pub fn usize end(ref u8[] data, usize start, u8 delimiter) {
    var usize i = start;
    while (i < data.len && data[i] != delimiter) { i += 1; }
    return i;
}
pub fn Result<i32, i32> id(ref u8[] raw) {
    if (raw.is_empty || raw.len > 10 || raw[0] < 49 || raw[0] > 57) { return .err(1); }
    var i32 value = 0;
    var usize i = 0;
    while (i < raw.len) {
        if (raw[i] < 48 || raw[i] > 57) { return .err(1); }
        let i32 digit = (raw[i] as i32) - 48;
        if (value > 214748364 || (value == 214748364 && digit > 7)) { return .err(1); }
        value = value * 10 + digit;
        i += 1;
    }
    return .ok(value);
}
pub fn bool title(ref u8[] raw) {
    if (raw.is_empty || raw.len > 512) { return false; }
    var bool content = false;
    var usize i = 0;
    while (i < raw.len) {
        if (raw[i] < 32 || raw[i] == 127) { return false; }
        if (raw[i] != 32) { content = true; }
        i += 1;
    }
    if (!content) { return false; }
    return switch (text::validate_utf8(raw)) { case .ok(_) : { true } case .err(_) : { false } };
}

// Canonical UTF-8 TSV: FYTASK1<TAB>next<LF>, sorted id<TAB>0|1<TAB>title<LF>.
@effects(console)
pub fn Result<i32, i32> validate(ref str path, ref u8[] data) {
    if (data.len > 262144) { return .err(output::error(path, "store exceeds 262144 bytes")); }
    let usize header_end = end(data, 0, 10);
    if (header_end == data.len || header_end < 9 || !bytes::equal(data[..8], b"FYTASK1\t")) {
        return .err(output::error(path, "invalid task store header"));
    }
    let i32 next = switch (id(data[8..header_end])) {
        case .ok(n) : { n } case .err(_) : { return .err(output::error(path, "invalid next task ID")); }
    };
    var usize start = header_end + 1;
    var usize count = 0;
    var i32 prior = 0;
    while (start < data.len) {
        if (count >= 256) { return .err(output::error(path, "task count exceeds 256")); }
        let usize line_end = end(data, start, 10);
        if (line_end == data.len) { return .err(output::error(path, "task record must end with LF")); }
        let usize tab = end(data, start, 9);
        if (tab >= line_end || line_end - tab < 4 || data[tab + 2] != 9 ||
            (data[tab + 1] != 48 && data[tab + 1] != 49)) {
            return .err(output::error(path, "invalid task record fields"));
        }
        let i32 current = switch (id(data[start..tab])) {
            case .ok(n) : { n } case .err(_) : { return .err(output::error(path, "invalid task ID in store")); }
        };
        if (current <= prior || current >= next) { return .err(output::error(path, "task IDs must increase and be below next ID")); }
        if (!title(data[tab + 3..line_end])) { return .err(output::error(path, "invalid task title in store")); }
        prior = current;
        count += 1;
        start = line_end + 1;
    }
    return .ok(next);
}

@effects(console)
pub fn Result<bytes::Buffer, i32> change(ref str path, ref u8[] data, i32 next, u32 command, ref u8[] argument, i32 requested, ref mut bytes::Buffer response) {
    // command 1 add, 2 done, 3 delete. All source data has already passed validate.
    var usize start = end(data, 0, 10) + 1;
    var usize count = 0;
    var bool found = false;
    while (start < data.len) {
        let usize tab = end(data, start, 9);
        let i32 current = switch (id(data[start..tab])) { case .ok(n) : { n } case .err(_) : { return .err(1); } };
        if (current == requested) { found = true; }
        count += 1;
        start = end(data, start, 10) + 1;
    }
    if (command == 1 && count >= 256) { return .err(output::error(path, "task count exceeds 256")); }
    if (command == 1 && next == 2147483647) { return .err(output::error(path, "task ID space exhausted")); }
    if (command != 1 && !found) { return .err(output::error(path, "task ID does not exist")); }
    let i32 future = if (command == 1) { next + 1 } else { next };
    var bytes::Buffer changed = bytes::empty();
    let bool a = buffer_ok(bytes::append(ref mut changed, b"FYTASK1\t"));
    let bool b = buffer_ok(bytes::append_i32(ref mut changed, future));
    let bool c = buffer_ok(bytes::append_byte(ref mut changed, 10));
    if (!a || !b || !c) { return .err(output::error(path, "cannot allocate store buffer")); }
    start = end(data, 0, 10) + 1;
    while (start < data.len) {
        let usize line_end = end(data, start, 10);
        let usize tab = end(data, start, 9);
        let i32 current = switch (id(data[start..tab])) { case .ok(n) : { n } case .err(_) : { return .err(1); } };
        if (command == 2 && current == requested) {
            let bool x = buffer_ok(bytes::append(ref mut changed, data[start..tab + 1]));
            let bool y = buffer_ok(bytes::append(ref mut changed, b"1\t"));
            let bool z = buffer_ok(bytes::append(ref mut changed, data[tab + 3..line_end + 1]));
            if (!x || !y || !z) { return .err(output::error(path, "cannot allocate store buffer")); }
        } else if (!(command == 3 && current == requested)) {
            if (!buffer_ok(bytes::append(ref mut changed, data[start..line_end + 1]))) { return .err(output::error(path, "cannot allocate store buffer")); }
        }
        start = line_end + 1;
    }
    if (command == 1) {
        let bool x = buffer_ok(bytes::append_i32(ref mut changed, next));
        let bool y = buffer_ok(bytes::append(ref mut changed, b"\t0\t"));
        let bool z = buffer_ok(bytes::append(ref mut changed, argument));
        let bool w = buffer_ok(bytes::append_byte(ref mut changed, 10));
        if (!x || !y || !z || !w) { return .err(output::error(path, "cannot allocate store buffer")); }
    }
    let bool x = buffer_ok(bytes::append(response, if (command == 1) { b"added id=" } else if (command == 2) { b"completed id=" } else { b"deleted id=" }));
    let bool y = buffer_ok(bytes::append_i32(response, if (command == 1) { next } else { requested }));
    let bool z = buffer_ok(bytes::append_byte(response, 10));
    if (!x || !y || !z) { return .err(output::error(path, "cannot allocate response")); }
    return .ok(changed);
}
