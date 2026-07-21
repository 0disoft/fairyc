module main;

@link_name("fairyc_fixture_add_one")
unsafe extern C fn i32 add_one(i32 value) ;

fn i32 main() {
    return unsafe { add_one(41) };
}
