module main;

fn i32 invalid(bool flag) {
    let Result<i32, i32> pending = .ok(42);
    if (flag) {
        return 1;
    };
    discard pending;
    return 0;
}
