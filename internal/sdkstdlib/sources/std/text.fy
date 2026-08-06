module std::text;

import std::bytes as bytes;

@repr(C)
struct raw_text_view {
    ptr u8 data;
    usize length;
}

@repr(C)
struct raw_text_result {
    u32 status;
    u32 allocation_status;
}

/// Reports the stable text and allocation statuses returned by the target runtime.
pub struct TextError {
    u32 status;
    u32 allocation_status;
}

/// Owns valid UTF-8 bytes and releases them automatically at scope exit.
pub resource String {
    ptr mut void data;
    usize capacity;
    usize alignment;
    usize length;

    cleanup {
        let raw_text_result dropped = unsafe {
            string_drop_storage_raw(ref mut this as ptr mut void)
        };
    }
}

/// Owns bytes terminated by one NUL byte with no interior NUL.
pub resource CString {
    ptr mut void data;
    usize capacity;
    usize alignment;
    usize storage_length;

    cleanup {
        let raw_text_result dropped = unsafe {
            cstring_drop_storage_raw(ref mut this as ptr mut void)
        };
    }
}

@link_name("fy_std_string_copy_storage_v0")
unsafe extern C fn raw_text_result string_copy_storage_raw(ptr raw_text_view input, ptr mut void output) ;

@link_name("fy_std_text_utf8_valid_v0")
unsafe extern C fn u32 utf8_valid_raw(ptr u8 data, usize length) ;

@link_name("fy_std_string_append_storage_v0")
unsafe extern C fn raw_text_result string_append_storage_raw(ptr mut void value, ptr raw_text_view suffix) ;

@link_name("fy_std_string_drop_storage_v0")
unsafe extern C fn raw_text_result string_drop_storage_raw(ptr mut void value) ;

@link_name("fy_std_string_from_buffer_storage_v0")
unsafe extern C fn raw_text_result string_from_buffer_storage_raw(ptr mut void input, ptr mut void output) ;

@link_name("fy_std_cstring_from_text_storage_v0")
unsafe extern C fn raw_text_result cstring_from_text_storage_raw(ptr raw_text_view input, ptr mut void output) ;

@link_name("fy_std_cstring_drop_storage_v0")
unsafe extern C fn raw_text_result cstring_drop_storage_raw(ptr mut void value) ;

fn raw_text_view raw_view(ref str value) {
    return raw_text_view {
        .data = unsafe { value as ptr u8 },
        .length = value.len,
    };
}

fn TextError text_error(raw_text_result result) {
    return TextError {
        .status = result.status,
        .allocation_status = result.allocation_status,
    };
}

fn TextError invalid_utf8_error() {
    return TextError {
        .status = 2,
        .allocation_status = 0,
    };
}

fn String empty_string() {
    return String {
        .data = null,
        .capacity = 0,
        .alignment = 0,
        .length = 0,
    };
}

fn CString empty_cstring() {
    return CString {
        .data = null,
        .capacity = 0,
        .alignment = 0,
        .storage_length = 0,
    };
}

fn i32 discard_string(String value) {
    return 0;
}

fn i32 discard_cstring(CString value) {
    return 0;
}

fn i32 discard_bytes_buffer(bytes::Buffer value) {
    return 0;
}

/// Validates bytes as UTF-8 and returns an allocation-free view with the source borrow origin.
pub fn Result<ref str, TextError> validate_utf8(ref u8[] value) {
    let u32 valid = unsafe {
        utf8_valid_raw(value as ptr u8, value.len)
    };
    return if (valid == 1) {
        .ok(unsafe {
            str::from_raw_parts(value, value as ptr u8, value.len)
        })
    } else {
        .err(invalid_utf8_error())
};
}

/// Borrows valid UTF-8 text as the same immutable bytes without allocation or copying.
pub fn ref u8[] as_bytes(ref str value) {
    return unsafe {
        byte_slice::from_raw_parts(value, value as ptr u8, value.len)
    };
}

/// Copies valid UTF-8 text into a newly owned String.
pub fn Result<String, TextError> copy(ref str value) {
    var String output = empty_string();
    let raw_text_view input = raw_view(value);
    let raw_text_result result = unsafe {
        string_copy_storage_raw(ref input as ptr raw_text_view, ref mut output as ptr mut void)
    };
    return if (result.status == 0) {
        .ok(output)
    } else {
        let i32 discarded = discard_string(output);
        .err(text_error(result))
    };
}

/// Converts an owned Buffer into a String without copying when its bytes are valid UTF-8.
/// Invalid UTF-8 returns the original Buffer unchanged with its storage intact.
pub fn Result<String, bytes::Buffer> from_buffer(bytes::Buffer value) {
    var bytes::Buffer owned = value;
    var String output = empty_string();
    let raw_text_result result = unsafe {
        string_from_buffer_storage_raw(ref mut owned as ptr mut void, ref mut output as ptr mut void)
    };
    return if (result.status == 0) {
        let i32 discarded = discard_bytes_buffer(owned);
        .ok(output)
    } else {
        let i32 discarded = discard_string(output);
        .err(owned)
    };
}

/// Appends valid UTF-8 text and returns the new byte length.
pub fn Result<usize, TextError> append(ref mut String value, ref str suffix) {
    let raw_text_view input = raw_view(suffix);
    let raw_text_result result = unsafe {
        string_append_storage_raw(value as ptr mut void, ref input as ptr raw_text_view)
    };
    return if (result.status == 0) {
        .ok(value.length)
    } else {
        .err(text_error(result))
    };
}

/// Returns the UTF-8 byte length.
pub fn usize string_len(ref String value) {
    return value.length;
}

/// Reports whether the String contains no bytes.
pub fn bool string_is_empty(ref String value) {
    return value.length == 0;
}

/// Borrows the owned UTF-8 bytes as immutable text.
pub fn ref str string_as_str(ref String value) {
    return unsafe {
        str::from_raw_parts(value, value.data, value.length)
    };
}

/// Copies text into a newly owned NUL-terminated CString.
pub fn Result<CString, TextError> cstring_from_text(ref str value) {
    var CString output = empty_cstring();
    let raw_text_view input = raw_view(value);
    let raw_text_result result = unsafe {
        cstring_from_text_storage_raw(ref input as ptr raw_text_view, ref mut output as ptr mut void)
    };
    return if (result.status == 0) {
        .ok(output)
    } else {
        let i32 discarded = discard_cstring(output);
        .err(text_error(result))
    };
}

/// Returns the byte length excluding the terminating NUL.
pub fn usize cstring_len(ref CString value) {
    return value.storage_length - 1;
}

/// Reports whether the CString contains only its terminating NUL.
pub fn bool cstring_is_empty(ref CString value) {
    return value.storage_length == 1;
}

/// Borrows the owned NUL-terminated bytes as an immutable CStr.
pub fn ref CStr cstring_as_cstr(ref CString value) {
    return unsafe {
        CStr::from_raw_parts(value, value.data, value.storage_length)
    };
}

/// Returns the stable text status code carried by error.
pub fn u32 error_status(TextError error) {
    return error.status;
}

/// Returns the stable allocation status code carried by error.
pub fn u32 error_allocation_status(TextError error) {
    return error.allocation_status;
}
