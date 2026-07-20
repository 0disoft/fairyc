module examples::windows_quickstart::main;

fn main() -> i32 {
    let bytes: &[u8] = b"*";
    if bytes.len > 0 { bytes[0] as i32 } else { 1 }
}
