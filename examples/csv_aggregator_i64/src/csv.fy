module csv;

import std::bytes as bytes;
import std::text as text;
import numbers as numbers;
import diagnostics as diag;

struct State {
    bool header;
    usize columns;
    usize selected;
    usize count;
    i64 sum;
    i64 minimum;
    i64 maximum;
    bool failed;
}

pub fn bool equals(ref u8[] left, ref u8[] right) {
    if (left.len != right.len) { return false; }
    var usize index = 0;
    while (index < left.len) {
        if (left[index] != right[index]) { return false; }
        index += 1;
    }
    return true;
}

// Decoded fields are terminated by NUL; original NUL bytes are explicitly rejected.
fn ref u8[] field(ref u8[] record, usize requested) {
    var usize start = 0;
    var usize index = 0;
    var usize current = 0;
    while (index < record.len) {
        if (record[index] == 0) {
            if (current == requested) { return record[start..index]; }
            current += 1;
            start = index + 1;
        }
        index += 1;
    }
    return record[..0];
}

fn bool buffer_ok(Result<usize, bytes::BufferError> result) {
    return switch (result) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
}

fn Result<usize, i32> push_character(ref mut bytes::Buffer output, u8 value, usize length) {
    if (length >= 16384) { return .err(1); }
    return switch (bytes::append_byte(output, value)) {
        case .ok(_) : { .ok(length + 1) }
        case .err(_) : { .err(2) }
    };
}

@effects(console)
fn Result<usize, i32> select_header(ref str path, usize line, ref u8[] row, usize columns, ref u8[] name) {
    var usize selected = columns;
    var usize index = 0;
    while (index < columns) {
        let ref u8[] key = field(row, index);
        if (key.is_empty) { return .err(diag::error(path, 1, line, "empty header")); }
        var usize previous = 0;
        while (previous < index) {
            if (equals(key, field(row, previous))) { return .err(diag::error(path, 1, line, "duplicate header")); }
            previous += 1;
        }
        if (equals(key, name)) { selected = index; }
        index += 1;
    }
    if (selected == columns) { return .err(diag::error(path, 1, line, "requested column not found")); }
    return .ok(selected);
}

@effects(console)
fn State finish_record(ref str path, usize number, usize line, ref u8[] row, usize columns, ref u8[] name, State initial) {
    var State state = initial;
    let bool utf8 = switch (text::validate_utf8(row)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
    if (!utf8) {
        let i32 code = diag::error(path, number, line, "invalid UTF-8");
        state.failed = true;
        return state;
    }
    if (!state.header) {
        state.selected = switch (select_header(path, line, row, columns, name)) {
            case .ok(selected) : { selected }
            case .err(_) : { state.failed = true; 0 }
        };
        state.columns = columns;
        state.header = true;
        return state;
    }
    if (columns != state.columns) {
        let i32 code = diag::error(path, number, line, "column count differs from header");
        state.failed = true;
        return state;
    }
    let i64 value = switch (numbers::parse(field(row, state.selected))) {
        case .ok(number) : { number }
        case .err(reason) : {
            let ref str message = if (reason == 1) { "empty numeric value" } else if (reason == 2) {
                "selected value must be a signed decimal integer"
            } else { "selected integer is outside i64 range" };
            let i32 code = diag::error(path, number, line, message);
            state.failed = true;
            0
        }
    };
    if (state.failed) { return state; }
    if (!numbers::sum_fits(state.sum, value)) {
        let i32 code = diag::error(path, number, line, "sum exceeds i64 range");
        state.failed = true;
        return state;
    }
    if (state.count == 0 || value < state.minimum) { state.minimum = value; }
    if (state.count == 0 || value > state.maximum) { state.maximum = value; }
    state.count = state.count + 1;
    state.sum = state.sum + value;
    return state;
}

@effects(console)
pub fn i32 aggregate(ref str path, ref u8[] input, ref u8[] name) {
    var State state = State {
        .header = false, .columns = 0, .selected = 0, .count = 0,
        .sum = 0, .minimum = 0, .maximum = 0, .failed = false,
    };
    var bytes::Buffer decoded = bytes::empty();
    var usize index = 0;
    if (input.len >= 3 && input[0] == 239 && input[1] == 187 && input[2] == 191) { index = 3; }
    var usize record_start = index;
    var usize record_number = 1;
    var usize record_line = 1;
    var usize line = 1;
    var usize columns = 1;
    var usize field_size = 0;
    // 0: start of field; 1: unquoted; 2: quoted; 3: after closing quote.
    var u32 mode = 0;
    while (index < input.len) {
        if (index - record_start >= 65536) {
            return diag::error(path, record_number, line, "record exceeds 65536 raw bytes");
        }
        let u8 character = input[index];
        if (character == 0) { return diag::error(path, record_number, line, "NUL byte is not supported"); }
        var bool append = false;
        if (mode == 2) {
            if (character == 34) { mode = 3; } else { append = true; }
        } else if (mode == 3 && character == 34) {
            mode = 2;
            append = true;
        } else if (character == 44 || character == 10 || character == 13) {
            var usize width = 1;
            if (character == 13) {
                if (index + 1 >= input.len || input[index + 1] != 10) {
                    return diag::error(path, record_number, line, "bare CR outside quoted field");
                }
                width = 2;
                if (index - record_start + width > 65536) {
                    return diag::error(path, record_number, line, "record exceeds 65536 raw bytes");
                }
            }
            if (!buffer_ok(bytes::append_byte(ref mut decoded, 0))) {
                return diag::error(path, record_number, line, "cannot allocate record buffer");
            }
            mode = 0;
            field_size = 0;
            if (character == 44) {
                if (columns == 64) { return diag::error(path, record_number, line, "record exceeds 64 columns"); }
                columns += 1;
            } else {
                state = finish_record(path, record_number, record_line, bytes::view(ref decoded), columns, name, state);
                if (state.failed) { return 1; }
                if (!buffer_ok(bytes::clear(ref mut decoded))) {
                    return diag::error(path, record_number, line, "cannot clear record buffer");
                }
                record_number += 1;
                line += 1;
                record_line = line;
                record_start = index + width;
                columns = 1;
            }
            index += width;
            continue;
        } else if (mode == 3) {
            return diag::error(path, record_number, line, "unexpected character after closing quote");
        } else if (character == 34) {
            if (mode != 0) { return diag::error(path, record_number, line, "quote inside unquoted field"); }
            mode = 2;
        } else {
            mode = 1;
            append = true;
        }
        if (append) {
            field_size = switch (push_character(ref mut decoded, character, field_size)) {
                case .ok(length) : { length }
                case .err(reason) : {
                    let ref str message = if (reason == 1) { "field exceeds 16384 decoded bytes" } else { "cannot allocate record buffer" };
                    return diag::error(path, record_number, line, message);
                }
            };
            if (character == 10) { line += 1; }
        }
        index += 1;
    }
    if (mode == 2) { return diag::error(path, record_number, line, "unterminated quoted field"); }
    if (record_start < input.len) {
        if (!buffer_ok(bytes::append_byte(ref mut decoded, 0))) {
            return diag::error(path, record_number, line, "cannot allocate record buffer");
        }
        state = finish_record(path, record_number, record_line, bytes::view(ref decoded), columns, name, state);
        if (state.failed) { return 1; }
    }
    if (!state.header) { return diag::error(path, 0, 0, "empty CSV input"); }
    if (state.count == 0) { return diag::error(path, 0, 0, "no data records"); }
    return diag::summary(state.count, state.sum, state.minimum, state.maximum);
}
