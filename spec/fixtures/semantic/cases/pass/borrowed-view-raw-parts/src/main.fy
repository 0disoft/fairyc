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

fn ref str relay(ref String value) {
    return as_str(value);
}
