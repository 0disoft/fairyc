module main;
fn i32 increment_bytes(ref mut u8[] view) {
    var usize index = 0;
    var i32 total = 0;
    for (let u8 value in view) {
        total += value as i32;
        if (index == 0) { view[1] = 9; }
        if (value != 255) {
            view[index] = value + 1;
            assert(view[index] != value);
        }
        index += 1;
    }
    return total;
}
fn void clear_wide(ref mut i64[] view) {
    var usize index = 0;
    for (let i64 value in view) {
        view[index] = 0;
        if (index == 0) { assert(value == 4000000000); }
        if (index == 1) { assert(value == -7); }
        if (index == 2) { assert(value == 9223372036854775807); }
        if (index == 3) { assert(value == -9223372036854775808); }
        index += 1;
    }
    assert(index == 4);
}
fn i64 first(ref mut i64[] view) {
    for (let i64 value in view) { view[0] = 8; return value; }
    return 0;
}
fn Result<i64,i32> step(i64 value) {
    if (value == -1) { return .err(5); }
    return .ok(value);
}
fn Result<i64,i32> propagate(ref mut i64[] view) {
    for (let i64 value in view) { let i64 unused = step(value)?; }
    return .ok(0);
}
fn i32 main() {
    var u8[4] bytes = [1,2,3,255];
    let ref mut u8[] writable = ref mut bytes;
    assert(increment_bytes(writable) == 268);
    assert(bytes[0] == 2 && bytes[1] == 10 && bytes[2] == 4 && bytes[3] == 255);
    var i64[4] wide = [4000000000,-7,9223372036854775807,-9223372036854775808];
    let ref mut i64[] values = ref mut wide;
    clear_wide(values);
    assert(wide[0] == 0 && wide[1] == 0 && wide[2] == 0 && wide[3] == 0);
    let ref mut i64[] again = ref mut wide;
    let ref mut i64[] empty = again[4..];
    for (let i64 value in empty) { assert(false); }
    let ref mut i64[] middle = again[1..3];
    var usize visits = 0;
    for (let i64 value in middle) {
        assert(value == 0);
        middle[visits] = 7;
        visits += 1;
        if (visits == 1) { continue; }
        break;
    }
    assert(visits == 2 && again[1] == 7 && again[2] == 7);
    let ref mut i64[] one = again[1..2];
    var usize nested = 0;
    for (let i64 value in one) {
        for (let i64 value in one) { assert(value == 7); nested += 1; }
        assert(value == 7);
    }
    assert(nested == 1);
    assert(first(one) == 7);
    assert(wide[1] == 8);
    wide[1] = -1;
    let ref mut i64[] errors = ref mut wide;
    let Result<i64,i32> result = propagate(errors);
    assert(switch (result) { case .ok(_): { false } case .err(code): { code == 5 } });
    wide[1] = 9;
    assert(wide[1] == 9);
    return 0;
}
