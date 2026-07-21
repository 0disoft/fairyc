module main;

fn i32 valid() {
    let Result<i32, i32> pending = .ok(42);
    discard pending;
    return 0;
}
