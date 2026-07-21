module main;

fn ref i32 relay(ref i32 left, ref i32 right) {
    return second(right, left);
}

fn ref i32 second(ref i32 left, ref i32 right) {
    return right;
}
