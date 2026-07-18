module examples::run::main;

fn answer(enabled: bool) -> Result<i32, i32> {
    if enabled {
        Result::Ok(20 + 22)
    } else {
        Result::Err(-1)
    }
}

fn main() -> i32 {
    match answer(true && 20 < 22) {
        Result::Ok(value) => value,
        Result::Err(error) => error,
    }
}
