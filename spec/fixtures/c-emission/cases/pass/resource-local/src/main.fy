module main;

resource Pair {
    left: i32;
    right: i32;

    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    let pair: Pair = Pair { left: 1, right: 42 };
    pair.right
}
