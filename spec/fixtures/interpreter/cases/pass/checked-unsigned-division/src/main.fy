module main;

fn divide() -> i32 {
    let byte_input: u8 = 84;
    let byte: u8 = byte_input / 2;
    let word_input: u32 = 85;
    let word: u32 = word_input % 43;
    let size_input: usize = 84;
    let size: usize = size_input / 2;
    if byte == 42 && word == 42 && size == 42 { 42 } else { 1 }
}

fn u8_zero() -> i32 {
    let numerator: u8 = 42;
    let zero: u8 = 0;
    let value: u8 = numerator / zero;
    42
}

fn u32_zero() -> i32 {
    let numerator: u32 = 42;
    let zero: u32 = 0;
    let value: u32 = numerator % zero;
    42
}

fn usize_zero() -> i32 {
    let numerator: usize = 42;
    let zero: usize = 0;
    let value: usize = numerator / zero;
    42
}
