module main;

struct ConsoleError {
    status: u32;
    written: usize;
}

fn write(accept: bool) -> Result<usize, ConsoleError> {
    if accept {
        Result::Ok(42)
    } else {
        Result::Err(ConsoleError { status: 3, written: 7 })
    }
}

fn main() -> i32 {
    match write(false) {
        Result::Ok(_) => 0,
        Result::Err(error) => if error.status == 3 { 42 } else { 0 },
    }
}
