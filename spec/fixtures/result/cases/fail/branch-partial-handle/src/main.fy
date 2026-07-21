module main;

fn i32 invalid(Result<i32, i32> input, bool flag) {
    if (flag) {
        discard input;
    } else {
        0;
    };
    return 0;
}
