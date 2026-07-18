module main;

fn invalid(flag: bool) -> i32 {
    let pending: Result<i32, i32> = Result::Ok(42);
    if flag {
        return 1;
    };
    discard pending;
    0
}
