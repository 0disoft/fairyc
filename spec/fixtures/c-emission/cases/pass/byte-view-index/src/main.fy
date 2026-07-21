module main;

fn i32 main() {
    let u8 value = b"\x00A"[1];
    return if (value == 65) { 42 } else { 1 };
}
