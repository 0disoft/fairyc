module main;

import std::bytes as bytes;

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

fn i32 main() {
    let i32 empty_code = switch (bytes::with_capacity(0)) {
        case .ok(buffer) : { check_empty(buffer) }
        case .err(_) : { 5 }
    };
    return if (empty_code != 0) {
        empty_code
    } else {
        switch (bytes::copy(b"Fairy")) {
            case .ok(buffer) : { check_buffer(buffer) }
            case .err(error) : { if (bytes::error_status(error) == 0) { 2 } else { 3 } }
        }
    };
}
