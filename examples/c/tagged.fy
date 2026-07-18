module examples::c::tagged;

fn choose() -> Option<i32> {
    Option::Some(41)
}

fn increment(input: Result<i32, i32>) -> Result<i32, i32> {
    let value: i32 = input?;
    Result::Ok(value + 1)
}

fn main() -> i32 {
    let base: i32 = match choose() {
        Option::Some(value) => value,
        Option::None => 0,
    };
    match increment(Result::Ok(base)) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
