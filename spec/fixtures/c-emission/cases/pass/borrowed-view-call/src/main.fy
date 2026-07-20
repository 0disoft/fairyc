module main;

fn text_length(text: &str) -> usize {
    text.len
}

fn byte_length(bytes: &[u8]) -> usize {
    bytes.len
}

fn main() -> i32 {
    if text_length("λx") == 3 && byte_length(b"\x00ABC") == 4 { 42 } else { 1 }
}
