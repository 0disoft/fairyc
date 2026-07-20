module main;

fn iterate() -> i32 {
    let bytes: &[u8] = b"\x13\x17";
    var index: usize = 0;
    var total: i32 = 0;
    while index < bytes.len {
        total = total + (bytes[index] as i32);
        index = index + 1;
    }
    total
}

fn u8_multiply_overflow() -> i32 {
    let value: u8 = 128;
    let overflow: u8 = value * 2;
    42
}

fn u32_subtract_underflow() -> i32 {
    let value: u32 = 0;
    let underflow: u32 = value - 1;
    42
}

fn usize_add_overflow() -> i32 {
    let value: usize = 18446744073709551615;
    let overflow: usize = value + 1;
    42
}
