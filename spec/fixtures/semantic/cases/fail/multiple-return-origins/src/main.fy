module main;

fn ref i32 choose(bool flag, ref i32 left, ref i32 right) {
    return if (flag) { left } else { right };
}
