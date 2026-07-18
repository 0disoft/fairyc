module main;

fn unwrap(input: Result<i32, i32>) -> i32 {
    match input {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
