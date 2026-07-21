module main;

fn i32 read(Region scratch, i32 value) {
    let ref i32 stored = scratch.place(value);
    return *stored;
}
