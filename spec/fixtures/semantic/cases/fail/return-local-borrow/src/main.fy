module main;

fn ref i32 invalid() {
    let i32 value = 1;
    return ref value;
}

fn ref i32 invalid_parameter(i32 input) {
    return ref input;
}
