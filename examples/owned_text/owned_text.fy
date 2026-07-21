module examples::owned_text::owned_text;

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

pub struct TextError {
    u32 status;
    u32 allocation_status;
}

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

@link_name("fy_std_string_append_storage_v0")
unsafe extern C fn raw_text_result string_append_storage_raw(ptr mut void value, ptr raw_text_view suffix) ;

@link_name("fy_std_string_drop_storage_v0")
unsafe extern C fn raw_text_result string_drop_storage_raw(ptr mut void value) ;

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

fn i32 discard_string(String value) {
    return 0;
}

fn i32 discard_cstring(CString value) {
    return 0;
}

pub fn Result<String, TextError> copy(ref str value) {
    var String output = String {
        .data = null,
        .capacity = 0,
        .alignment = 0,
        .length = 0,
    };
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

pub fn usize string_len(ref String value) {
    return value.length;
}

pub fn ref str string_as_str(ref String value) {
    return unsafe {
        str::from_raw_parts(value, value.data, value.length)
    };
}

pub fn Result<CString, TextError> cstring_from_text(ref str value) {
    var CString output = CString {
        .data = null,
        .capacity = 0,
        .alignment = 0,
        .storage_length = 0,
    };
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

pub fn usize cstring_len(ref CString value) {
    return value.storage_length - 1;
}

pub fn ref CStr cstring_as_cstr(ref CString value) {
    return unsafe {
        CStr::from_raw_parts(value, value.data, value.storage_length)
    };
}

pub fn u32 error_status(TextError error) {
    return error.status;
}

pub fn u32 error_allocation_status(TextError error) {
    return error.allocation_status;
}
