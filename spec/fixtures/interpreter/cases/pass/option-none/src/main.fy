module main;

fn choose() -> Option<i32> {
    Option::None
}

fn main() -> i32 {
    match choose() {
        Option::Some(value) => value,
        Option::None => 42,
    }
}
