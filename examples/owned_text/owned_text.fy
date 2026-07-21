module examples::owned_text::owned_text;

@repr(C)
struct raw_text_view {
    data: *u8;
    length: usize;
}

@repr(C)
struct raw_text_result {
    status: u32;
    allocation_status: u32;
}

pub struct TextError {
    status: u32;
    allocation_status: u32;
}

pub resource String {
    data: *mut void;
    capacity: usize;
    alignment: usize;
    length: usize;

    drop fn(self: &mut Self) {
        let dropped: raw_text_result = unsafe {
            string_drop_storage_raw(&mut self as *mut void)
        };
    }
}

pub resource CString {
    data: *mut void;
    capacity: usize;
    alignment: usize;
    storage_length: usize;

    drop fn(self: &mut Self) {
        let dropped: raw_text_result = unsafe {
            cstring_drop_storage_raw(&mut self as *mut void)
        };
    }
}

@link_name("fy_std_string_copy_storage_v0")
unsafe extern C fn string_copy_storage_raw(
    input: *raw_text_view,
    output: *mut void
) -> raw_text_result;

@link_name("fy_std_string_append_storage_v0")
unsafe extern C fn string_append_storage_raw(
    value: *mut void,
    suffix: *raw_text_view
) -> raw_text_result;

@link_name("fy_std_string_drop_storage_v0")
unsafe extern C fn string_drop_storage_raw(value: *mut void) -> raw_text_result;

@link_name("fy_std_cstring_from_text_storage_v0")
unsafe extern C fn cstring_from_text_storage_raw(
    input: *raw_text_view,
    output: *mut void
) -> raw_text_result;

@link_name("fy_std_cstring_drop_storage_v0")
unsafe extern C fn cstring_drop_storage_raw(value: *mut void) -> raw_text_result;

fn raw_view(value: &str) -> raw_text_view {
    raw_text_view {
        data: unsafe { value as *u8 },
        length: value.len,
    }
}

fn text_error(result: raw_text_result) -> TextError {
    TextError {
        status: result.status,
        allocation_status: result.allocation_status,
    }
}

fn discard_string(value: String) -> i32 {
    0
}

fn discard_cstring(value: CString) -> i32 {
    0
}

pub fn copy(value: &str) -> Result<String, TextError> {
    var output: String = String {
        data: null,
        capacity: 0,
        alignment: 0,
        length: 0,
    };
    let input: raw_text_view = raw_view(value);
    let result: raw_text_result = unsafe {
        string_copy_storage_raw(&input as *raw_text_view, &mut output as *mut void)
    };
    if result.status == 0 {
        Result::Ok(output)
    } else {
        let discarded: i32 = discard_string(output);
        Result::Err(text_error(result))
    }
}

pub fn append(value: &mut String, suffix: &str) -> Result<usize, TextError> {
    let input: raw_text_view = raw_view(suffix);
    let result: raw_text_result = unsafe {
        string_append_storage_raw(value as *mut void, &input as *raw_text_view)
    };
    if result.status == 0 {
        Result::Ok(value.length)
    } else {
        Result::Err(text_error(result))
    }
}

pub fn string_len(value: &String) -> usize {
    value.length
}

pub fn string_as_str(value: &String) -> &str {
    unsafe {
        str::from_raw_parts(value, value.data, value.length)
    }
}

pub fn cstring_from_text(value: &str) -> Result<CString, TextError> {
    var output: CString = CString {
        data: null,
        capacity: 0,
        alignment: 0,
        storage_length: 0,
    };
    let input: raw_text_view = raw_view(value);
    let result: raw_text_result = unsafe {
        cstring_from_text_storage_raw(&input as *raw_text_view, &mut output as *mut void)
    };
    if result.status == 0 {
        Result::Ok(output)
    } else {
        let discarded: i32 = discard_cstring(output);
        Result::Err(text_error(result))
    }
}

pub fn cstring_len(value: &CString) -> usize {
    value.storage_length - 1
}

pub fn cstring_as_cstr(value: &CString) -> &CStr {
    unsafe {
        CStr::from_raw_parts(value, value.data, value.storage_length)
    }
}

pub fn error_status(error: TextError) -> u32 {
    error.status
}

pub fn error_allocation_status(error: TextError) -> u32 {
    error.allocation_status
}
