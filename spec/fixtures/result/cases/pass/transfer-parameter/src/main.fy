module main;

fn i32 consume(Result<i32, i32> input) {
    discard input;
    return 0;
}

fn i32 valid() {
    return consume(.ok(42));
}
