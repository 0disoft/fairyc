module main;

fn i32 in_bounds() {
    let u8 value = b"\x00A"[1];
    return if (value == 65) { 42 } else { 1 };
}

fn i32 out_of_bounds() {
    let u8 value = b"A"[1];
    return 42;
}
