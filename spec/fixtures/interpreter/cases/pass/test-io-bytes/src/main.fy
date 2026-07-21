module main;

fn i32 write_stderr_byte(i32 value) {
    return value;
}

fn i32 write_stdout_byte(i32 value) {
    return value;
}

fn i32 main() {
    let i32 first = write_stdout_byte(52);
    let i32 second = write_stderr_byte(69);
    let i32 third = write_stdout_byte(50);
    let i32 fourth = write_stderr_byte(10);
    return first + second + third + fourth + 38;
}
