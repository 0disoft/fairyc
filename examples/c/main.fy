module examples::c::main;

fn add(left: i32, right: i32) -> i32 {
    left + right
}

fn main() -> i32 {
    var count: i32 = 0;
    while count < 3 {
        count = count + 1;
    }
    add(count, 39)
}
