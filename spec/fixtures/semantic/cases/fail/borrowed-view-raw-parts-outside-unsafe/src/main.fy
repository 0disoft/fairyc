module main;

resource String {
    ptr mut void data;
    usize length;

    cleanup {}
}

fn ref str as_str(ref String value) {
    return str::from_raw_parts(value, value.data, value.length);
}
