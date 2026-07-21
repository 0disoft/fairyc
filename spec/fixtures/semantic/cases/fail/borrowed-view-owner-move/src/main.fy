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

fn consume_owned(value: String) -> i32 {
    0
}

fn main() -> i32 {
    let value: String = String { data: null, length: 0 };
    let view: &str = as_str(&value);
    let consumed: i32 = consume_owned(value);
    if view.len == 0 { 0 } else { 1 }
}
