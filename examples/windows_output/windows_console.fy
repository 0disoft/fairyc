module examples::windows_output::windows_console;

@repr(C)
struct write_result {
    status: u32;
    written: usize;
}

pub struct ConsoleError {
    status: u32;
    written: usize;
}

@effects(console)
@link_name("fy_std_windows_stdout_write_all_v0")
unsafe extern C fn stdout_write_all(data: *u8, length: usize) -> write_result;

@effects(console)
pub fn stdout_contract_edge_states() -> bool {
    let empty: write_result = unsafe {
        stdout_write_all(null, 0)
    };
    let invalid: write_result = unsafe {
        stdout_write_all(null, 1)
    };
    empty.status == 0 && empty.written == 0 &&
        invalid.status == 1 && invalid.written == 0
}

@effects(console)
pub fn stdout_write_text(text: &str) -> Result<usize, ConsoleError> {
    let result: write_result = unsafe {
        stdout_write_all(text as *u8, text.len)
    };
    if result.status == 0 {
        Result::Ok(result.written)
    } else {
        Result::Err(ConsoleError {
            status: result.status,
            written: result.written,
        })
    }
}

pub fn byte_view_ready(bytes: &[u8]) -> bool {
    let first: u8 = bytes[0];
    let second: u8 = bytes[1];
    first == 0 && second == 65 && bytes.len == 2
}
