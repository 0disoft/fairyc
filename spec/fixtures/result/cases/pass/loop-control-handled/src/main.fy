module main;

fn break_path(flag: bool) -> i32 {
    var pending: Result<i32, i32> = Result::Ok(0);
    discard pending;
    while flag {
        pending = Result::Ok(1);
        break;
    }
    discard pending;
    0
}
fn continue_path(flag: bool) -> i32 {
    var pending: Result<i32, i32> = Result::Ok(0);
    discard pending;
    while flag {
        pending = Result::Ok(1);
        discard pending;
        continue;
    }
    discard pending;
    0
}
