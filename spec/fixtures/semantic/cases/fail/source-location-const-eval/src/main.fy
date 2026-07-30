module main;

@const_eval
fn i32 bad() {
    return source_location().len as i32;
}

const i32 BAD = bad();

fn i32 main() {
    return 0;
}
