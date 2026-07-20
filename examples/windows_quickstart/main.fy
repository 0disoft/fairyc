module examples::windows_quickstart::main;

fn main() -> i32 {
    let bytes: &[u8] = b"\x00\x13\x17\x00"[1..];
    var index: usize = 0b0;
    var total: i32 = 0x0;
    while index < bytes.len {
        total = total + (bytes[index] as i32);
        index = index + 1;
    }
    assert(total == 0x2A);
    assert(-0x8000_0000 + 0x7FFF_FFFF + 0b1 == 0);
    let answers: [u8; 3] = [40, 41, 42];
    let flags: [bool; 2] = [false, true];
    let signed: [i32; 1] = [42];
    let wide: [u32; 1] = [42];
    let offsets: [usize; 1] = [2];
    assert(answers[2] as i32 == total);
    assert(flags[1]);
    assert(signed[0] == total);
    assert(wide[0] == 42);
    assert(offsets[0] == 2);
    if !bytes.is_empty && b"".is_empty { total } else { 0b1 }
}
