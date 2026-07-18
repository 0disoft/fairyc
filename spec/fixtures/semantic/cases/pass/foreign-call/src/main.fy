module main;

unsafe extern C fn add_one(value: i32) -> i32;

fn main() -> i32 {
    unsafe { add_one(41) }
}
