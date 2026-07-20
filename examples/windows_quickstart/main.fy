module examples::windows_quickstart::main;

fn second(left: &[u8], right: &[u8]) -> &[u8] {
    right
}

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
    let answer_bytes: &[u8] = &answers;
    let backup: [u8; 3] = [1, 2, 3];
    var selected_bytes: &[u8] = &backup;
    selected_bytes = &answers;
    var loop_selected: &[u8] = &backup;
    var loop_count: usize = 0;
    while loop_count < 1 {
        loop_selected = &answers;
        loop_count = loop_count + 1;
    }
    var decoy: [u8; 3] = [4, 5, 6];
    let selected_by_call: &[u8] = second(&decoy, &answers);
    decoy = [7, 8, 9];
    assert(answers.len == 3);
    assert(flags.len == 2);
    assert(signed.len == 1);
    assert(wide.len == 1);
    assert(offsets.len == 1);
    assert(answer_bytes.len == 3);
    assert(answer_bytes[0] == 40);
    assert(selected_bytes[2] == 42);
    assert(loop_selected[2] == 42);
    assert(selected_by_call[2] == 42);
    assert(answers[2] as i32 == total);
    assert(flags[1]);
    assert(signed[0] == total);
    assert(wide[0] == 42);
    assert(offsets[0] == 2);
    if !bytes.is_empty && b"".is_empty { total } else { 0b1 }
}
