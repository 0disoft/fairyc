module main;

fn increment(input: Result<i32, i32>) -> Result<i32, i32> {
    let value: i32 = input?;
    Result::Ok(value + 1)
}

fn main() -> i32 {
    match increment(Result::Ok(41)) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
