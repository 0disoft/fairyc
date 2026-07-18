module main;

fn invalid(flag: bool) -> i32 {
    while flag {
        let pending: Result<i32, i32> = Result::Ok(1);
        break;
        discard pending;
    }
    0
}
