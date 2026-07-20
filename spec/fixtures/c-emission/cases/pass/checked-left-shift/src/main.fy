module main;

fn main() -> i32 {
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
