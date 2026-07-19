module main;

@link_name("fairyc_fixture_make_c_size_t")
unsafe extern C fn make_c_size_t() -> c::size_t;

@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn consume_c_size_t(value: c::size_t) -> i32;

fn main() -> i32 {
    unsafe {
        consume_c_size_t(make_c_size_t())
    }
}
