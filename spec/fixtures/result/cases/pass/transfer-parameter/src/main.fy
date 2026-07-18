module main;

fn consume(input: Result<i32, i32>) -> i32 {
    discard input;
    0
}

fn valid() -> i32 {
    consume(Result::Ok(42))
}
