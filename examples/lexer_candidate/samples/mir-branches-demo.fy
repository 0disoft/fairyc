module main;

fn i32 main() {
    var i32 value = 5;
    let i32 selected = if (value > 0) { value + 2 } else { 0 };
    assert(selected == 7 && (true || fail()));
    return selected;
}

fn bool fail() {
    assert(false);
    return false;
}
