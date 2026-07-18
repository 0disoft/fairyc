module main;

fn invalid() -> i32 {
    let pending: Result<i32, i32> = Result::Ok(42);
    0
}
