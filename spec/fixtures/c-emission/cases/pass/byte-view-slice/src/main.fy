module main;

fn main() -> i32 {
    let all: &[u8] = b"fairy"[..];
    let bytes: &[u8] = all[1..4];
    if bytes.len == 3 && bytes[0] == 97 && bytes[2] == 114 { 42 } else { 1 }
}
