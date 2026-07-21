module main;

fn i32 main() {
    let u8 byte = b"*"[0];
    let u32 word = byte as u32;
    let usize size = word as usize;
    let ptr u8 pointer = null;
    return if (true == true && -1 < 0 && byte > 41 && word >= 42 && size <= 42 && pointer == null) { 42 } else { 1 };
}
