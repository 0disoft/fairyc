module main;

fn main() -> i32 {
    let result: Result<i32, i32> = Result::Err(-42);
    match result {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
