module main;

import std::bytes as bytes;

fn bool view_equals(ref u8[] left, ref u8[] right) {
    if (left.len != right.len) {
        return false;
    }
    var usize index = 0;
    while (index < left.len) {
        if (left[index] != right[index]) {
            return false;
        }
        index += 1;
    }
    return true;
}

fn bool append_succeeded(Result<usize, bytes::BufferError> result) {
    return switch (result) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
}

fn i32 check_empty(bytes::Buffer buffer) {
    let ref u8[] empty = bytes::view(ref buffer)[..];
    return if (bytes::len(ref buffer) == 0 &&
        bytes::capacity(ref buffer) == 0 &&
        bytes::is_empty(ref buffer) && empty.is_empty) {
        0
    } else {
        4
    };
}

fn i32 check_buffer(bytes::Buffer input) {
    var bytes::Buffer buffer = input;
    let bool copied = bytes::len(ref buffer) == 5 &&
        bytes::capacity(ref buffer) >= 5 &&
        !bytes::is_empty(ref buffer);
    let bool reserved = switch (bytes::reserve(ref mut buffer, 16)) {
        case .ok(capacity) : { capacity >= 16 }
        case .err(_) : { false }
    };
    let bool appended = switch (bytes::append(ref mut buffer, b"C")) {
        case .ok(length) : { length == 6 }
        case .err(_) : { false }
    };
    let bool appended_byte = switch (bytes::append_byte(ref mut buffer, 33)) {
        case .ok(length) : { length == 7 }
        case .err(_) : { false }
    };
    let bool resized = switch (bytes::resize_zeroed(ref mut buffer, 9)) {
        case .ok(length) : { length == 9 }
        case .err(_) : { false }
    };
    let ref mut u8[] writable = bytes::view_mut(ref mut buffer);
    let bool zeroed = writable[7] == 0 && writable[8] == 0;
    writable[7] = 42;
    writable[8] = 43;
    let bool mutated = writable[7] == 42 && writable[8] == 43;
    let ref u8[] readable = bytes::view(ref buffer);
    let bool contents = readable.len == 9 &&
        readable[0] == 70 && readable[1] == 97 && readable[2] == 105 &&
        readable[3] == 114 && readable[4] == 121 && readable[5] == 67 &&
        readable[6] == 33 && readable[7] == 42 && readable[8] == 43;
    let bool truncated = switch (bytes::truncate(ref mut buffer, 4)) {
        case .ok(length) : { length == 4 }
        case .err(_) : { false }
    };
    let bool cleared = switch (bytes::clear(ref mut buffer)) {
        case .ok(length) : { length == 0 && bytes::is_empty(ref buffer) }
        case .err(_) : { false }
    };
    return if (copied && reserved && appended && appended_byte && resized && zeroed && mutated && contents && truncated && cleared) {
        0
    } else {
        1
    };
}

fn i32 check_formatting(bytes::Buffer input) {
    var bytes::Buffer buffer = input;
    if (!append_succeeded(bytes::append(ref mut buffer, b"i32="))) {
        return 6;
    }
    if (!append_succeeded(bytes::append_i32(ref mut buffer, 0))) {
        return 7;
    }
    if (!append_succeeded(bytes::append(ref mut buffer, b","))) {
        return 8;
    }
    if (!append_succeeded(bytes::append_i32(ref mut buffer, -2147483648))) {
        return 9;
    }
    if (!append_succeeded(bytes::append(ref mut buffer, b","))) {
        return 10;
    }
    if (!append_succeeded(bytes::append_i32(ref mut buffer, 2147483647))) {
        return 11;
    }
    if (!append_succeeded(bytes::append(ref mut buffer, b";u32="))) {
        return 12;
    }
    if (!append_succeeded(bytes::append_u32(ref mut buffer, 0))) {
        return 13;
    }
    if (!append_succeeded(bytes::append(ref mut buffer, b","))) {
        return 14;
    }
    if (!append_succeeded(bytes::append_u32(ref mut buffer, 4294967295))) {
        return 15;
    }
    if (!append_succeeded(bytes::append(ref mut buffer, b";usize="))) {
        return 16;
    }
    if (!append_succeeded(bytes::append_usize(ref mut buffer, 0))) {
        return 17;
    }
    if (!append_succeeded(bytes::append(ref mut buffer, b","))) {
        return 18;
    }
    if (!append_succeeded(bytes::append_usize(ref mut buffer, 18446744073709551615))) {
        return 19;
    }
    if (!append_succeeded(bytes::append(ref mut buffer, b";bool="))) {
        return 20;
    }
    if (!append_succeeded(bytes::append_bool(ref mut buffer, true))) {
        return 21;
    }
    if (!append_succeeded(bytes::append(ref mut buffer, b","))) {
        return 22;
    }
    if (!append_succeeded(bytes::append_bool(ref mut buffer, false))) {
        return 23;
    }
    let ref u8[] actual = bytes::view(ref buffer);
    return if (view_equals(actual, b"i32=0,-2147483648,2147483647;u32=0,4294967295;usize=0,18446744073709551615;bool=true,false")) {
        0
    } else {
        24
    };
}

fn i32 main() {
    let i32 empty_code = switch (bytes::with_capacity(0)) {
        case .ok(buffer) : { check_empty(buffer) }
        case .err(_) : { 5 }
    };
    return if (empty_code != 0) {
        empty_code
    } else {
        let i32 buffer_code = switch (bytes::copy(b"Fairy")) {
            case .ok(buffer) : { check_buffer(buffer) }
            case .err(error) : { if (bytes::error_status(error) == 0) { 2 } else { 3 } }
        };
        if (buffer_code != 0) {
            buffer_code
        } else {
            let bytes::Buffer formatting = bytes::empty();
            check_formatting(formatting)
        }
    };
}
