module main;

fn store(scratch: Region, value: i32) -> &i32 {
    scratch.place(value)
}
