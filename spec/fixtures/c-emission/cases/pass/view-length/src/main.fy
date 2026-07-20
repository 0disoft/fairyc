module main;

fn main() -> i32 {
    let text: &str = "λx";
    let bytes: &[u8] = b"\x00ABC";
    if text.len == 3 && bytes.len == 4 { 42 } else { 1 }
}
