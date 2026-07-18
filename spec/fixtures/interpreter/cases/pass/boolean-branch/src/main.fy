module main;

fn choose(flag: bool) -> i32 {
    if flag { 42 } else { 0 }
}

fn main() -> i32 {
    choose(true)
}
