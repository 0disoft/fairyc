module main;

fn i32 valid() {
    var Result<i32, i32> pending = .ok(1);
    discard pending;
    pending = .ok(2);
    discard pending;
    return 0;
}
