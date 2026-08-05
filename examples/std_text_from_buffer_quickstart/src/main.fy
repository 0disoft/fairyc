module main;

import std::bytes as bytes;
import std::text as text;

fn i32 check_string(text::String value) {
    let ref str view = text::string_as_str(ref value);
    return if (view.len == 5 && !view.is_empty && text::string_len(ref value) == 5) {
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

fn i32 main() {
    let i32 valid_code = switch (bytes::copy(b"Fairy")) {
        case .ok(value) : {
            switch (text::from_buffer(value)) {
                case .ok(string) : { check_string(string) }
                case .err(_) : { 1 }
            }
        }
        case .err(_) : { 2 }
    };
    let i32 invalid_code = switch (bytes::copy(b"\xff")) {
        case .ok(value) : {
            switch (text::from_buffer(value)) {
                case .ok(_) : { 3 }
                case .err(returned) : { check_returned_buffer(returned) }
            }
        }
        case .err(_) : { 4 }
    };
    let i32 empty_code = switch (bytes::copy(b"")) {
        case .ok(value) : {
            switch (text::from_buffer(value)) {
                case .ok(string) : { if (text::string_is_empty(ref string)) { 0 } else { 5 } }
                case .err(_) : { 6 }
            }
        }
        case .err(_) : { 7 }
    };
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
