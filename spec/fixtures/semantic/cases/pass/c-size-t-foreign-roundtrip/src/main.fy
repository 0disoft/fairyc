module main;

@link_name("fairyc_fixture_make_c_size_t")
unsafe extern C fn c::size_t make_c_size_t() ;

@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn i32 consume_c_size_t(c::size_t value) ;

fn i32 main() {
    return unsafe {
        consume_c_size_t(make_c_size_t())
    };
}
