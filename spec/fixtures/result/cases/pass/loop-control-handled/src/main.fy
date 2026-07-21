module main;

fn i32 break_path(bool flag) {
    var Result<i32, i32> pending = .ok(0);
    discard pending;
    while (flag) {
        pending = .ok(1);
        break;
    }
    discard pending;
    return 0;
}

fn i32 continue_path(bool flag) {
    var Result<i32, i32> pending = .ok(0);
    discard pending;
    while (flag) {
        pending = .ok(1);
        discard pending;
        continue;
    }
    discard pending;
    return 0;
}
