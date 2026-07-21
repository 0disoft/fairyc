module main;

fn i32 read(ref i32 value) {
    return *value;
}

fn i32 update() {
    var i32 value = 1;
    let ref i32 view = ref value;
    let i32 before = read(view);
    value = 2;
    return before + value;
}
