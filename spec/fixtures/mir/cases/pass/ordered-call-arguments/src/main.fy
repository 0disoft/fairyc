module main;

fn first() -> i32 {
    1
}

fn second() -> i32 {
    2
}

fn choose(left: i32, right: i32) -> i32 {
    left
}

fn run() -> i32 {
    choose(first(), second())
}
