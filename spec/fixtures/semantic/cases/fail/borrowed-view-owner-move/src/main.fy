module main;

resource String {
    ptr mut void data;
    usize length;

    cleanup {}
}

fn ref str as_str(ref String value) {
    return unsafe {
        str::from_raw_parts(value, value.data, value.length)
    };
}

fn i32 consume_owned(String value) {
    return 0;
}

fn i32 main() {
    let String value = String { .data = null, .length = 0 };
    let ref str view = as_str(ref value);
    let i32 consumed = consume_owned(value);
    return if (view.len == 0) { 0 } else { 1 };
}
