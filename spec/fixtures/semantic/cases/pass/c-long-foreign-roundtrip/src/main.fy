module main;

@link_name("fairyc_fixture_make_c_long")
unsafe extern C fn c::long make_c_long() ;

@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn i32 consume_c_long(c::long value) ;

fn i32 main() {
    return unsafe { consume_c_long(make_c_long()) };
}
