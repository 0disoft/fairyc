module main;

fn ref i32 select(bool flag, ref i32 left, ref i32 right) {
    var ref i32 current = left;
    if (flag) { current = right; 0 } else { current = right; 0 };
    return current;
}
