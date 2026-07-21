module main;

fn i32 invalid(bool flag) {
    while (flag) {
        let Result<i32, i32> pending = .ok(1);
        break;
        discard pending;
    }
    return 0;
}
