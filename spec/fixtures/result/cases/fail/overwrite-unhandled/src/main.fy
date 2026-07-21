module main;

fn i32 invalid() {
    var Result<i32, i32> pending = .ok(1);
    pending = .ok(2);
    discard pending;
    return 0;
}
