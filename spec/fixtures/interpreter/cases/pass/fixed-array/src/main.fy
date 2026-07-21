module main;

fn u8 pick(u8[3] values, usize index) {
    return values[index];
}

fn u8 view_pick(ref u8[] values, usize index) {
    return values[index];
}

fn i32 main() {
    let u8[3] values = [40, 41, 42];
    let bool[2] flags = [false, true];
    let i32[1] signed = [42];
    let u32[1] wide = [42];
    let usize[1] offsets = [2];
    return if (values.len == 3 && flags.len == 2 && signed.len == 1 && wide.len == 1 && offsets.len == 1 &&
       flags[1] && signed[0] == 42 && wide[0] == 42 && offsets[0] == 2 && view_pick(ref values, 0) == 40) {
        pick(values, offsets[0]) as i32
    } else {
        1
    };
}

fn i32 bounds_trap() {
    let i32[2] values = [40, 42];
    return values[2];
}
