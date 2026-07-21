module main;

fn ref u8[] second(ref u8[] left, ref u8[] right) {
    return right;
}

fn i32 main() {
    let u8[1] left = [1];
    var u8[1] right = [42];
    let ref u8[] selected = second(ref left, ref right);
    right = [0];
    return selected[0] as i32;
}
