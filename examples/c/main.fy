module examples::c::main;

fn i32 add(i32 left, i32 right) {
    return left + right;
}

fn i32 main() {
    var i32 count = 0;
    while (count < 3) {
        count = count + 1;
    }
    return add(count, 39);
}
