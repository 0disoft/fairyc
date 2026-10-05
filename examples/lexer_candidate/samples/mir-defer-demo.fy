module main;

fn i32 main() {
    var i32 value = 65;
    defer check_root(value);
    value = 1;
    discard { defer check_nested(66); };
    return 7;
}
fn void check_root(i32 value) { assert(value == 65); }
fn void check_nested(i32 value) { assert(value == 66); }
