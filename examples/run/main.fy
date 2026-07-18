module examples::run::main;

fn answer(enabled: bool) -> i32 {
    if enabled {
        20 + 22
    } else {
        0
    }
}

fn main() -> i32 {
    answer(true && 20 < 22)
}
