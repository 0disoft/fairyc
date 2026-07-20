module main;

fn evaluate() -> i32 {
    let byte_input: u8 = 84;
    let byte: u8 = byte_input >> 1;
    let word_input: u32 = 43;
    let word: u32 = word_input & 63;
    let word_shift_input: u32 = 84;
    let word_shift: u32 = word_shift_input >> 1;
    let size_input: usize = 43;
    let size: usize = (size_input ^ 1) | 0;
    let size_shift_input: usize = 84;
    let size_shift: usize = size_shift_input >> 1;
    if byte == 42 && word == 43 && word_shift == 42 && size == 42 && size_shift == 42 { 42 } else { 1 }
}

fn u8_invalid_shift() -> i32 {
    let value: u8 = 42;
    let amount: u8 = 8;
    let shifted: u8 = value >> amount;
    42
}

fn u32_invalid_shift() -> i32 {
    let value: u32 = 42;
    let amount: u32 = 32;
    let shifted: u32 = value >> amount;
    42
}

fn usize_invalid_shift() -> i32 {
    let value: usize = 42;
    let amount: usize = 64;
    let shifted: usize = value >> amount;
    42
}
