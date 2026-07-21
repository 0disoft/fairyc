module examples::owned_text::main;

import examples::owned_text::owned_text as text;

fn check_string(mut value: text::String) -> i32 {
    let initial_view: &str = text::string_as_str(&value);
    if initial_view.len != 5 {
        10
    } else {
        match text::append(&mut value, "Fairy") {
            Result::Ok(length) => if length == 10 && text::string_len(&value) == 10 {
                let appended_view: &str = text::string_as_str(&value);
                if appended_view.len == 10 { 0 } else { 13 }
            } else {
                11
            },
            Result::Err(_) => 12,
        }
    }
}

fn consume_cstr(value: &CStr) -> i32 {
    0
}

fn check_cstring(value: text::CString, expected: usize) -> i32 {
    let view: &CStr = text::cstring_as_cstr(&value);
    if text::cstring_len(&value) == expected { consume_cstr(view) } else { 20 }
}

fn implicit_string_drop(value: text::String) -> i32 {
    let view: &str = text::string_as_str(&value);
    if view.len == 4 { 0 } else { 30 }
}

fn implicit_cstring_drop(value: text::CString) -> i32 {
    let view: &CStr = text::cstring_as_cstr(&value);
    if text::cstring_len(&value) == 4 { consume_cstr(view) } else { 31 }
}

fn main() -> i32 {
    let string_code: i32 = match text::copy("Fairy") {
        Result::Ok(value) => check_string(value),
        Result::Err(_) => 1,
    };
    let string_drop_code: i32 = match text::copy("drop") {
        Result::Ok(value) => implicit_string_drop(value),
        Result::Err(_) => 2,
    };
    let cstring_code: i32 = match text::cstring_from_text("Fairy") {
        Result::Ok(value) => check_cstring(value, 5),
        Result::Err(_) => 3,
    };
    let empty_cstring_code: i32 = match text::cstring_from_text("") {
        Result::Ok(value) => check_cstring(value, 0),
        Result::Err(_) => 4,
    };
    let cstring_drop_code: i32 = match text::cstring_from_text("drop") {
        Result::Ok(value) => implicit_cstring_drop(value),
        Result::Err(_) => 5,
    };
    let interior_nul_code: i32 = match text::cstring_from_text("a\0b") {
        Result::Ok(_) => 6,
        Result::Err(error) => if text::error_status(error) == 3 { 0 } else { 7 },
    };
    if string_code != 0 {
        string_code
    } else if string_drop_code != 0 {
        string_drop_code
    } else if cstring_code != 0 {
        cstring_code
    } else if empty_cstring_code != 0 {
        empty_cstring_code
    } else if cstring_drop_code != 0 {
        cstring_drop_code
    } else if interior_nul_code != 0 {
        interior_nul_code
    } else {
        42
    }
}
