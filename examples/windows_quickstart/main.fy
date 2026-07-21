module examples::windows_quickstart::main;

fn ref u8[] second(ref u8[] left, ref u8[] right) {
    return right;
}

fn u8 set_last(ref mut u8[] bytes) {
    bytes[2] = 42;
    return bytes[2];
}

fn i32 main() {
    let ref u8[] bytes = b"\x00\x13\x17\x00"[1..];
    var usize index = 0b0;
    var i32 total = 0x0;
    while (index < bytes.len) {
        total = total + (bytes[index] as i32);
        index = index + 1;
    }
    assert(total == 0x2A);
    assert(-0x8000_0000 + 0x7FFF_FFFF + 0b1 == 0);
    var u8[3] answers = [0, 0, 0];
    answers[0] = 40;
    answers[1] = 41;
    let u8 changed = set_last(ref mut answers);
    let bool[2] flags = [false, true];
    let i32[1] signed = [42];
    let u32[1] wide = [42];
    let usize[1] offsets = [2];
    let ref u8[] answer_bytes = ref answers;
    let u8[3] backup = [1, 2, 3];
    var ref u8[] selected_bytes = ref backup;
    selected_bytes = ref answers;
    var ref u8[] loop_selected = ref backup;
    var usize loop_count = 0;
    while (loop_count < 1) {
        loop_selected = ref answers;
        loop_count = loop_count + 1;
    }
    var u8[3] decoy = [4, 5, 6];
    let ref u8[] selected_by_call = second(ref decoy, ref answers);
    decoy = [7, 8, 9];
    assert(answers.len == 3);
    assert(flags.len == 2);
    assert(signed.len == 1);
    assert(wide.len == 1);
    assert(offsets.len == 1);
    assert(answer_bytes.len == 3);
    assert(answer_bytes[0] == 40);
    assert(changed == 42);
    assert(selected_bytes[2] == 42);
    assert(loop_selected[2] == 42);
    assert(selected_by_call[2] == 42);
    assert(answers[2] as i32 == total);
    assert(flags[1]);
    assert(signed[0] == total);
    assert(wide[0] == 42);
    assert(offsets[0] == 2);
    return if (!bytes.is_empty && b"".is_empty) { total } else { 0b1 };
}
