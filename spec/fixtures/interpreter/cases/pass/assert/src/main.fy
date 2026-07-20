module main;

fn passes() -> i32 {
    assert(true);
    42
}

fn fails() -> i32 {
    assert(false);
    42
}
