module main;
fn T step<T>(T value) { return value + 1; }
fn T twice<T>(T value) {
    let T first = step<T>(value);
    return step<T>(first);
}
fn i32 main() { return twice<i32>(40); }
