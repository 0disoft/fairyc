module main;

fn main() -> i32 {
    let bytes: &[u8] = b"fairy"[1..4];
    if bytes.len == 3 && bytes[0] == 97 && bytes[2] == 114 { 42 } else { 1 }
}

fn empty_at_end() -> i32 {
    let bytes: &[u8] = b"fairy"[5..5];
    if bytes.is_empty { 42 } else { 1 }
}

fn start_after_end() -> i32 {
    let bytes: &[u8] = b"fairy"[4..2];
    42
}

fn end_after_length() -> i32 {
    let bytes: &[u8] = b"fairy"[0..6];
    42
}
