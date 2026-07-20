module examples::windows_output::main;

@repr(C)
struct write_result {
    status: u32;
    written: usize;
}

@link_name("fy_std_windows_stdout_write_all_v0")
unsafe extern C fn stdout_write_all(data: *u8, length: usize) -> write_result;

fn main() -> i32 {
    let text: &str = "Fairy C says hi\n";
    let empty: write_result = unsafe {
        stdout_write_all(null, 0)
    };
    let invalid: write_result = unsafe {
        stdout_write_all(null, 1)
    };
    let result: write_result = unsafe {
        stdout_write_all(text as *u8, 16)
    };
    let edge_states: bool = empty.status == 0 && empty.written == 0 &&
        invalid.status == 1 && invalid.written == 0;
    if edge_states && result.status == 0 && result.written == 16 { 42 } else { 1 }
}
