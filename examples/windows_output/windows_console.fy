module examples::windows_output::windows_console;

@repr(C)
struct write_result {
    status: u32;
    written: usize;
}

@link_name("fy_std_windows_stdout_write_all_v0")
unsafe extern C fn stdout_write_all(data: *u8, length: usize) -> write_result;

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

pub fn stdout_write_text(text: &str) -> bool {
    let result: write_result = unsafe {
        stdout_write_all(text as *u8, text.len)
    };
    result.status == 0 && result.written == text.len
}

pub fn byte_view_ready(bytes: &[u8]) -> bool {
    let first: u8 = bytes[0];
    let second: u8 = bytes[1];
    first == 0 && second == 65 && bytes.len == 2
}
