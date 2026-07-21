module main;

fn i32 invalid(bool flag) {
    while (flag) {
        let Result<i32, i32> pending = .ok(1);
        continue;
        discard pending;
    }
    return 0;
}
