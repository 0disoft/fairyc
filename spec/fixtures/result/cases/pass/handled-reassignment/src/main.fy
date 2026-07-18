module main;

fn valid() -> i32 {
    var pending: Result<i32, i32> = Result::Ok(1);
    discard pending;
    pending = Result::Ok(2);
    discard pending;
    0
}
