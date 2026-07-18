module examples::regions::main;

fn read(scratch: Region, value: i32) -> i32 {
    let stored: &i32 = scratch.place(value);
    *stored
}
