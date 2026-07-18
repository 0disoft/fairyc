module main;

fn choose() -> Option<i32> {
    Option::Some(42)
}

fn main() -> i32 {
    match choose() {
        Option::Some(value) => value,
        Option::None => 0,
    }
}
