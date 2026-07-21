module main;

@link_name("fairyc_fixture_make_c_int")
unsafe extern C fn c::int make_c_int() ;

@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn i32 consume_c_int(c::int value) ;

fn i32 main() {
    return unsafe {
        consume_c_int(make_c_int())
    };
}
