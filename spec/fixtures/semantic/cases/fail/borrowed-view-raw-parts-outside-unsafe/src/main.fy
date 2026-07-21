module main;

resource String {
    data: *mut void;
    length: usize;

    drop fn(self: &mut Self) {}
}

fn as_str(value: &String) -> &str {
    str::from_raw_parts(value, value.data, value.length)
}
