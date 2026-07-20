module main;

fn main() -> i32 {
    let value: u8 = b"\x00A"[1];
    if value == 65 { 42 } else { 1 }
}
