module main;

import std::bytes as bytes;
import std::text as text;

fn i32 check_round_trip(text::String value) {
    var bytes::Buffer buffer = text::into_buffer(value);
    let ref mut u8[] mutable = bytes::view_mut(ref mut buffer);
    mutable[0] = 0x66;
    let ref u8[] changed = bytes::view(ref buffer);
    return if (changed.len == 5 && changed[0] == 0x66 && changed[4] == 0x79) {
        0
    } else {
        10
    };
}

fn i32 check_returned_buffer(bytes::Buffer value) {
    let ref u8[] view = bytes::view(ref value);
    return if (bytes::len(ref value) == 1 && bytes::capacity(ref value) == 1 &&
        view.len == 1 && view[0] == 0xff) {
        0
    } else {
        20
    };
}

fn i32 check_empty_buffer(bytes::Buffer value) {
    return if (bytes::is_empty(ref value) && bytes::capacity(ref value) == 0) {
        0
    } else {
        5
    };
}

fn i32 check_valid_round_trip() {
    return switch (bytes::copy(b"Fairy")) {
        case .ok(value) : {
            switch (text::from_buffer(value)) {
                case .ok(string) : { check_round_trip(string) }
                case .err(_) : { 1 }
            }
        }
        case .err(_) : { 2 }
    };
}

fn i32 check_invalid_return() {
    return switch (bytes::copy(b"\xff")) {
        case .ok(value) : {
            switch (text::from_buffer(value)) {
                case .ok(_) : { 3 }
                case .err(returned) : { check_returned_buffer(returned) }
            }
        }
        case .err(_) : { 4 }
    };
}

fn i32 check_empty_round_trip() {
    return switch (bytes::copy(b"")) {
        case .ok(value) : {
            switch (text::from_buffer(value)) {
                case .ok(string) : { check_empty_buffer(text::into_buffer(string)) }
                case .err(_) : { 6 }
            }
        }
        case .err(_) : { 7 }
    };
}

fn i32 main() {
    let i32 valid_code = check_valid_round_trip();
    let i32 invalid_code = check_invalid_return();
    let i32 empty_code = check_empty_round_trip();
    return if (valid_code != 0) {
        valid_code
    } else if (invalid_code != 0) {
        invalid_code
    } else if (empty_code != 0) {
        empty_code
    } else {
        42
    };
}
