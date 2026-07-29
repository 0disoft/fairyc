module main;

fn i32 observe(i32 value) {
    return value;
}

fn void emit(i32 value) {
    discard observe(value);
}

fn i32 main() {
    var i32 first = 65;
    defer emit(first);
    first = 1;
    if (true) {
        defer emit(66);
    }
    return 42;
}
