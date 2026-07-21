module main;

@link_name("fairyc_fixture_make_c_ssize_t")
unsafe extern C fn c::ssize_t make_c_ssize_t() ;

@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn i32 consume_c_ssize_t(c::ssize_t value) ;

fn i32 main() {
    return unsafe {
        consume_c_ssize_t(make_c_ssize_t())
    };
}
