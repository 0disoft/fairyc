module main;

@link_name("fairyc_fixture_make_c_long")
unsafe extern C fn make_c_long() -> c::long;

@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn consume_c_long(value: c::long) -> i32;

fn main() -> i32 {
    unsafe { consume_c_long(make_c_long()) }
}
