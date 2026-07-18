module main;

fn main() -> i32 {
    let result: Result<i32, i32> = Result::Ok(42);
    match result {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
