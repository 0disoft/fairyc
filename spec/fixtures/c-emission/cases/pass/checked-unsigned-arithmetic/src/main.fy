module main;

fn main() -> i32 {
    let byte: u8 = 6;
    let byte_product: u8 = byte * 7;
    let word: u32 = 50;
    let word_difference: u32 = word - 8;
    let size: usize = 41;
    let size_sum: usize = size + 1;
    if byte_product == 42 && word_difference == 42 && size_sum == 42 { 42 } else { 1 }
}
