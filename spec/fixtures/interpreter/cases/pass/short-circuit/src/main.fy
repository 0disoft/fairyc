module main;

fn main() -> i32 {
    if (false && (1 / 0 == 0)) || (true || (1 / 0 == 0)) {
        42
    } else {
        0
    }
}
