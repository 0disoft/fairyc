module main;

fn ref i32 select(bool run, ref i32 left, ref i32 right) {
    var ref i32 current = left;
    while (run) {
        current = right;
        break;
    }
    return current;
}
