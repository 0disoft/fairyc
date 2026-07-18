module main;

fn main() -> i32 {
    let all: i32 = ~0;
    let low: i32 = all & 63;
    let mixed: i32 = low ^ 21;
    mixed | 0
}
