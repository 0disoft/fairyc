module main;

fn i32 main() {
    let ref u8[] all = b"fairy"[..];
    let ref u8[] bytes = all[1..4];
    return if (bytes.len == 3 && bytes[0] == 97 && bytes[2] == 114) { 42 } else { 1 };
}
