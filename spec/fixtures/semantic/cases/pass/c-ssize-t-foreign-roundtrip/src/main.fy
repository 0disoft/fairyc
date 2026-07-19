module main;

@link_name("fairyc_fixture_make_c_ssize_t")
unsafe extern C fn make_c_ssize_t() -> c::ssize_t;

@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn consume_c_ssize_t(value: c::ssize_t) -> i32;

fn main() -> i32 {
    unsafe {
        consume_c_ssize_t(make_c_ssize_t())
    }
}
