module main;

fn i32 valid(bool flag) {
    let Result<i32, i32> pending = if (flag) {
        .ok(42)
    } else {
        .err(7)
    };
    discard pending;
    return 0;
}
