module main;

fn i32 evaluate() {
    let i32 signed_input = 21;
    let i32 signed = signed_input << 1;
    let u8 byte_input = 21;
    let u8 byte = byte_input << 1;
    let u32 word_input = 21;
    let u32 word = word_input << 1;
    let usize size_input = 21;
    let usize size = size_input << 1;
    return if (signed == 42 && byte == 42 && word == 42 && size == 42) { 42 } else { 1 };
}

fn i32 i32_negative_amount() {
    let i32 value = 1073741824;
    return value << -1;
}

fn i32 u8_width_precedes_overflow() {
    let u8 value = 255;
    let u8 amount = 8;
    let u8 shifted = value << amount;
    return 42;
}

fn i32 i32_overflow() {
    let i32 value = 1073741824;
    return value << 1;
}

fn i32 u8_overflow() {
    let u8 value = 128;
    let u8 amount = 1;
    let u8 shifted = value << amount;
    return 42;
}
