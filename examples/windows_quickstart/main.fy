module examples::windows_quickstart::main;

fn main() -> i32 {
    let bytes: &[u8] = b"\x13\x17";
    var index: usize = 0;
    var total: i32 = 0;
    while index < bytes.len {
        total = total + (bytes[index] as i32);
        index = index + 1;
    }
    if !bytes.is_empty && b"".is_empty { total } else { 1 }
}
