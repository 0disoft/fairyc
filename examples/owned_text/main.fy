module examples::owned_text::main;

import examples::owned_text::owned_text as text;

fn i32 check_string(text::String input) {
    var text::String value = input;
    let ref str initial_view = text::string_as_str(ref value);
    return if (initial_view.len != 5) {
        10
    } else {
        switch (text::append(ref mut value, "Fairy")) {
            case .ok(length) : { if (length == 10 && text::string_len(ref value) == 10) {
                let ref str appended_view = text::string_as_str(ref value);
                if (appended_view.len == 10) { 0 } else { 13 }
            } else {
                11
            } }
            case .err(_) : { 12 }
        }
    };
}

fn i32 consume_cstr(ref CStr value) {
    return 0;
}

fn i32 check_cstring(text::CString value, usize expected) {
    let ref CStr view = text::cstring_as_cstr(ref value);
    return if (text::cstring_len(ref value) == expected) { consume_cstr(view) } else { 20 };
}

fn i32 implicit_string_drop(text::String value) {
    let ref str view = text::string_as_str(ref value);
    return if (view.len == 4) { 0 } else { 30 };
}

fn i32 implicit_cstring_drop(text::CString value) {
    let ref CStr view = text::cstring_as_cstr(ref value);
    return if (text::cstring_len(ref value) == 4) { consume_cstr(view) } else { 31 };
}

fn i32 main() {
    let i32 string_code = switch (text::copy("Fairy")) {
        case .ok(value) : { check_string(value) }
        case .err(_) : { 1 }
    };
    let i32 string_drop_code = switch (text::copy("drop")) {
        case .ok(value) : { implicit_string_drop(value) }
        case .err(_) : { 2 }
    };
    let i32 cstring_code = switch (text::cstring_from_text("Fairy")) {
        case .ok(value) : { check_cstring(value, 5) }
        case .err(_) : { 3 }
    };
    let i32 empty_cstring_code = switch (text::cstring_from_text("")) {
        case .ok(value) : { check_cstring(value, 0) }
        case .err(_) : { 4 }
    };
    let i32 cstring_drop_code = switch (text::cstring_from_text("drop")) {
        case .ok(value) : { implicit_cstring_drop(value) }
        case .err(_) : { 5 }
    };
    let i32 interior_nul_code = switch (text::cstring_from_text("a\0b")) {
        case .ok(_) : { 6 }
        case .err(error) : { if (text::error_status(error) == 3) { 0 } else { 7 } }
    };
    return if (string_code != 0) {
        string_code
    } else if (string_drop_code != 0) {
        string_drop_code
    } else if (cstring_code != 0) {
        cstring_code
    } else if (empty_cstring_code != 0) {
        empty_cstring_code
    } else if (cstring_drop_code != 0) {
        cstring_drop_code
    } else if (interior_nul_code != 0) {
        interior_nul_code
    } else {
        42
    };
}
