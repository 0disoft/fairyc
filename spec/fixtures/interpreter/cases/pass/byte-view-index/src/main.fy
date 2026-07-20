module main;

fn in_bounds() -> i32 {
    let value: u8 = b"\x00A"[1];
    42
}

fn out_of_bounds() -> i32 {
    let value: u8 = b"A"[1];
    42
}
