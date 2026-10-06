module main;

fn i32 main() {
    var i32[3] original = [1, 2, 3];
    let copy = original;
    original[0] = 9;
    assert(copy[0] == 1 && original[0] == 9 && copy.len == 3);
    return copy[2];
}
