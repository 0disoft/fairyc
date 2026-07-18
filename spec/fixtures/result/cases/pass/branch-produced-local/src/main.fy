module main;

fn valid(flag: bool) -> i32 {
    let pending: Result<i32, i32> = if flag {
        Result::Ok(42)
    } else {
        Result::Err(7)
    };
    discard pending;
    0
}
