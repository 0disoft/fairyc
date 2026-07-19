module main;

@link_name("fairyc_fixture_make_c_int")
unsafe extern C fn make_c_int() -> c::int;

@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn consume_c_int(value: c::int) -> i32;

fn main() -> i32 {
    unsafe {
        consume_c_int(make_c_int())
    }
}
