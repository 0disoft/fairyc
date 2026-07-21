module main;

fn ref i32 store(Region scratch, i32 value) {
    return scratch.place(value);
}
