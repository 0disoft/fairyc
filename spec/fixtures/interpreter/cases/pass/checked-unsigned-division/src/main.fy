module main;

fn i32 divide() {
    let u8 byte_input = 84;
    let u8 byte = byte_input / 2;
    let u32 word_input = 85;
    let u32 word = word_input % 43;
    let usize size_input = 84;
    let usize size = size_input / 2;
    return if (byte == 42 && word == 42 && size == 42) { 42 } else { 1 };
}

fn i32 u8_zero() {
    let u8 numerator = 42;
    let u8 zero = 0;
    let u8 value = numerator / zero;
    return 42;
}

fn i32 u32_zero() {
    let u32 numerator = 42;
    let u32 zero = 0;
    let u32 value = numerator % zero;
    return 42;
}

fn i32 usize_zero() {
    let usize numerator = 42;
    let usize zero = 0;
    let usize value = numerator / zero;
    return 42;
}
