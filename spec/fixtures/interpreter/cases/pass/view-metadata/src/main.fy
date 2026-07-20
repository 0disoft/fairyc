module main;

fn main() -> i32 {
    let text: &str = "fairy";
    let empty_text: &str = "";
    let bytes: &[u8] = b"fairy";
    let empty_bytes: &[u8] = b"";
    if text.len == 5 && bytes.len == 5 && !text.is_empty && empty_text.is_empty && !bytes.is_empty && empty_bytes.is_empty { 42 } else { 1 }
}
