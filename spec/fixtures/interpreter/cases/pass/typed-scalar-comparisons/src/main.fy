module main;

fn main() -> i32 {
    let byte: u8 = b"*"[0];
    let word: u32 = byte as u32;
    let size: usize = word as usize;
    let pointer: *u8 = null;
    if true == true && -1 < 0 && byte > 41 && word >= 42 && size <= 42 && pointer == null { 42 } else { 1 }
}
