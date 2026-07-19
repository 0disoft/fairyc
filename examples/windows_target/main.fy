module examples::windows_target::main;

@link_name("fairyc_unused_windows_long")
unsafe extern C fn unused_windows_long(value: c::long) -> c::long;

@link_name("fy_std_text_utf8_valid_v0")
unsafe extern C fn utf8_valid(data: *u8, length: usize) -> u32;

fn main() -> i32 {
    if unsafe { utf8_valid(null, 0) } == 1 { 42 } else { 1 }
}
