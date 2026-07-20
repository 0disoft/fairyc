module main;

fn in_bounds() -> i32 {
    let value: u8 = b"\x00A"[1];
    if value == 65 { 42 } else { 1 }
}

fn out_of_bounds() -> i32 {
    let value: u8 = b"A"[1];
    42
}
