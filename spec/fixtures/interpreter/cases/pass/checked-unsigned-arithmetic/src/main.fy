module main;

fn i32 iterate() {
    let ref u8[] bytes = b"\x13\x17";
    var usize index = 0;
    var i32 total = 0;
    while (index < bytes.len) {
        total = total + (bytes[index] as i32);
        index = index + 1;
    }
    return total;
}

fn i32 u8_multiply_overflow() {
    let u8 value = 128;
    let u8 overflow = value * 2;
    return 42;
}

fn i32 u32_subtract_underflow() {
    let u32 value = 0;
    let u32 underflow = value - 1;
    return 42;
}

fn i32 usize_add_overflow() {
    let usize value = 18446744073709551615;
    let usize overflow = value + 1;
    return 42;
}
