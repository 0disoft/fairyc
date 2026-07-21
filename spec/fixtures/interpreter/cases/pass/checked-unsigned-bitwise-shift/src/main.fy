module main;

fn i32 evaluate() {
    let u8 byte_input = 84;
    let u8 byte = byte_input >> 1;
    let u8 byte_not_input = 213;
    let u8 byte_not = ~byte_not_input;
    let u32 word_input = 43;
    let u32 word = word_input & 63;
    let u32 word_not_input = 4294967253;
    let u32 word_not = ~word_not_input;
    let u32 word_shift_input = 84;
    let u32 word_shift = word_shift_input >> 1;
    let usize size_input = 43;
    let usize size = (size_input ^ 1) | 0;
    let usize size_not_input = 18446744073709551573;
    let usize size_not = ~size_not_input;
    let usize size_shift_input = 84;
    let usize size_shift = size_shift_input >> 1;
    return if (byte == 42 && byte_not == 42 && word == 43 && word_not == 42 && word_shift == 42 && size == 42 && size_not == 42 && size_shift == 42) { 42 } else { 1 };
}

fn i32 u8_invalid_shift() {
    let u8 value = 42;
    let u8 amount = 8;
    let u8 shifted = value >> amount;
    return 42;
}

fn i32 u32_invalid_shift() {
    let u32 value = 42;
    let u32 amount = 32;
    let u32 shifted = value >> amount;
    return 42;
}

fn i32 usize_invalid_shift() {
    let usize value = 42;
    let usize amount = 64;
    let usize shifted = value >> amount;
    return 42;
}
