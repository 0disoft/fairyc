module main;

fn invalid(flag: bool) -> i32 {
    var pending: Result<i32, i32> = Result::Ok(0);
    discard pending;
    while flag {
        pending = Result::Ok(1);
        continue;
    }
    discard pending;
    0
}
