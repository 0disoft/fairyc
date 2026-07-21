module main;

fn u8 update(ref mut u8[] bytes) {
    bytes[1] = 42;
    return bytes[1];
}

fn i32 bounds_trap() {
    var u8[3] values = [1, 2, 3];
    values[3] = 42;
    return 0;
}

fn i32 main() {
    var u8[3] values = [1, 2, 3];
    values[0] = 40;
    let u8 changed = update(ref mut values);
    return if (values[0] == 40 && values[1] == changed && values[1] == 42) {
        42
    } else {
        1
    };
}
