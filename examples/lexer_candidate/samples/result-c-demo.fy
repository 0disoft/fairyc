module main;

fn Result<i64,bool> lookup(bool found) {
    return if(found) { .ok(4000000000) } else { .err(true) };
}
fn i64 read(Result<i64,bool> result) {
    return switch(result) {
        case .ok(value): { value }
        case .err(missing): { assert(missing); -1 }
    };
}
fn i32 main() {
    assert(read(lookup(true)) == 4000000000);
    assert(read(lookup(false)) == -1);
    return 0;
}
