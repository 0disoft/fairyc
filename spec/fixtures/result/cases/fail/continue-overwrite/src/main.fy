module main;

fn i32 invalid(bool flag) {
    var Result<i32, i32> pending = .ok(0);
    discard pending;
    while (flag) {
        pending = .ok(1);
        continue;
    }
    discard pending;
    return 0;
}
