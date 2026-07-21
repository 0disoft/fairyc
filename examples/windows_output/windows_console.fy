module examples::windows_output::windows_console;

@repr(C)
struct write_result {
    u32 status;
    usize written;
}

pub struct ConsoleError {
    u32 status;
    usize written;
}

@effects(console)
@link_name("fy_std_windows_stdout_write_all_v0")
unsafe extern C fn write_result stdout_write_all(ptr u8 data, usize length) ;

@effects(console)
pub fn bool stdout_contract_edge_states() {
    let write_result empty = unsafe {
        stdout_write_all(null, 0)
    };
    let write_result invalid = unsafe {
        stdout_write_all(null, 1)
    };
    return empty.status == 0 && empty.written == 0 &&
        invalid.status == 1 && invalid.written == 0;
}

@effects(console)
pub fn Result<usize, ConsoleError> stdout_write_text(ref str text) {
    let write_result result = unsafe {
        stdout_write_all(text as ptr u8, text.len)
    };
    return if (result.status == 0) {
        .ok(result.written)
    } else {
        .err(ConsoleError {
            .status = result.status,
            .written = result.written,
        })
    };
}

pub fn bool byte_view_ready(ref u8[] bytes) {
    let u8 first = bytes[0];
    let u8 second = bytes[1];
    return first == 0 && second == 65 && bytes.len == 2;
}
