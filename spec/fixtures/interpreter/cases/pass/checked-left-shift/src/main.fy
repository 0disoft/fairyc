module main;

fn evaluate() -> i32 {
    let signed_input: i32 = 21;
    let signed: i32 = signed_input << 1;
    let byte_input: u8 = 21;
    let byte: u8 = byte_input << 1;
    let word_input: u32 = 21;
    let word: u32 = word_input << 1;
    let size_input: usize = 21;
    let size: usize = size_input << 1;
    if signed == 42 && byte == 42 && word == 42 && size == 42 { 42 } else { 1 }
}

fn i32_negative_amount() -> i32 {
    let value: i32 = 1073741824;
    value << -1
}

fn u8_width_precedes_overflow() -> i32 {
    let value: u8 = 255;
    let amount: u8 = 8;
    let shifted: u8 = value << amount;
    42
}

fn i32_overflow() -> i32 {
    let value: i32 = 1073741824;
    value << 1
}

fn u8_overflow() -> i32 {
    let value: u8 = 128;
    let amount: u8 = 1;
    let shifted: u8 = value << amount;
    42
}
