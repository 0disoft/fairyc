module main;

import std::text as text;

fn i32 consume_cstr(ref CStr value) {
    return 0;
}

fn i32 check_string(text::String input) {
    var text::String value = input;
    let ref str initial = text::string_as_str(ref value);
    let ref u8[] initial_bytes = text::as_bytes(initial);
    let bool initial_ok = initial.len == 5 &&
        initial_bytes.len == 5 &&
        initial_bytes[0] == 70 && initial_bytes[1] == 97 &&
        initial_bytes[2] == 105 && initial_bytes[3] == 114 &&
        initial_bytes[4] == 121 &&
        text::string_len(ref value) == 5 &&
        !text::string_is_empty(ref value);
    return if (!initial_ok) {
        10
    } else {
        switch (text::append(ref mut value, " C")) {
            case .ok(length) : {
                let ref str appended = text::string_as_str(ref value);
                let ref u8[] appended_bytes = text::as_bytes(appended);
                if (length == 7 && appended.len == 7 && appended_bytes.len == 7 &&
                    appended_bytes[5] == 32 && appended_bytes[6] == 67 &&
                    text::string_len(ref value) == 7) {
                    0
                } else {
                    11
                }
            }
            case .err(_) : { 12 }
        }
    };
}

fn i32 check_cstring(text::CString value, usize expected, bool empty) {
    let ref CStr view = text::cstring_as_cstr(ref value);
    return if (text::cstring_len(ref value) == expected &&
        text::cstring_is_empty(ref value) == empty) {
        consume_cstr(view)
    } else {
        20
    };
}

fn i32 main() {
    let i32 valid_utf8_code = switch (text::validate_utf8(b"Fairy \xce\xbb")) {
        case .ok(value) : {
            let ref u8[] bytes = text::as_bytes(value);
            if (value.len == 8 && !value.is_empty && bytes.len == 8 &&
                bytes[5] == 32 && bytes[6] == 206 && bytes[7] == 187) {
                0
            } else {
                8
            }
        }
        case .err(_) : { 9 }
    };
    let i32 invalid_utf8_code = switch (text::validate_utf8(b"\xff")) {
        case .ok(_) : { 13 }
        case .err(error) : {
            if (text::error_status(error) == 2 && text::error_allocation_status(error) == 0) {
                0
            } else {
                14
            }
        }
    };
    let i32 string_code = switch (text::copy("Fairy")) {
        case .ok(value) : { check_string(value) }
        case .err(_) : { 1 }
    };
    let i32 empty_string_code = switch (text::copy("")) {
        case .ok(value) : {
            let ref str view = text::string_as_str(ref value);
            let ref u8[] bytes = text::as_bytes(view);
            if (text::string_is_empty(ref value) && bytes.is_empty) { 0 } else { 2 }
        }
        case .err(_) : { 3 }
    };
    let i32 cstring_code = switch (text::cstring_from_text("Fairy")) {
        case .ok(value) : { check_cstring(value, 5, false) }
        case .err(_) : { 4 }
    };
    let i32 empty_cstring_code = switch (text::cstring_from_text("")) {
        case .ok(value) : { check_cstring(value, 0, true) }
        case .err(_) : { 5 }
    };
    let i32 interior_nul_code = switch (text::cstring_from_text("a\0b")) {
        case .ok(_) : { 6 }
        case .err(error) : {
            if (text::error_status(error) == 3 && text::error_allocation_status(error) == 0) {
                0
            } else {
                7
            }
        }
    };
    return if (valid_utf8_code != 0) {
        valid_utf8_code
    } else if (invalid_utf8_code != 0) {
        invalid_utf8_code
    } else if (string_code != 0) {
        string_code
    } else if (empty_string_code != 0) {
        empty_string_code
    } else if (cstring_code != 0) {
        cstring_code
    } else if (empty_cstring_code != 0) {
        empty_cstring_code
    } else if (interior_nul_code != 0) {
        interior_nul_code
    } else {
        42
    };
}
