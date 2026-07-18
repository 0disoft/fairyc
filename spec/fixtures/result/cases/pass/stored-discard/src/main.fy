module main;

fn valid() -> i32 {
    let pending: Result<i32, i32> = Result::Ok(42);
    discard pending;
    0
}
