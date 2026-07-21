module main;

resource String {
    data: *mut void;
    length: usize;

    drop fn(self: &mut Self) {}
}

fn as_str(value: &String) -> &str {
    unsafe {
        str::from_raw_parts(value, value.data, value.length)
    }
}

fn relay(value: &String) -> &str {
    as_str(value)
}
