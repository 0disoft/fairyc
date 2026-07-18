module main;

fn write_stderr_byte(value: i32) -> i32 {
    value
}

fn write_stdout_byte(value: i32) -> i32 {
    value
}

fn main() -> i32 {
    let first: i32 = write_stdout_byte(52);
    let second: i32 = write_stderr_byte(69);
    let third: i32 = write_stdout_byte(50);
    let fourth: i32 = write_stderr_byte(10);
    first + second + third + fourth + 38
}
