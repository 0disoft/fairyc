module main;

fn void record(i32 value) {
    assert(value == 42);
}

fn i32 main() {
    var i32 value = 40;
    defer record(value + 2);
    value = 0;
    return 42;
}
