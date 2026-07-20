module main;

fn main() -> i32 {
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
