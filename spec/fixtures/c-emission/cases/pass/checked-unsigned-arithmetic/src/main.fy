module main;

fn i32 main() {
    let u8 byte = 6;
    let u8 byte_product = byte * 7;
    let u32 word = 50;
    let u32 word_difference = word - 8;
    let usize size = 41;
    let usize size_sum = size + 1;
    return if (byte_product == 42 && word_difference == 42 && size_sum == 42) { 42 } else { 1 };
}
