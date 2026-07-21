module main;

fn ref i32 choose(bool flag, ref i32 left, ref i32 right) {
    var ref i32 current = left;
    if (flag) { current = left; 0 } else { current = right; 0 };
    return current;
}
