module examples::windows_target::main;

@link_name("fairyc_unused_windows_long")
unsafe extern C fn unused_windows_long(value: c::long) -> c::long;

fn main() -> i32 {
    42
}
