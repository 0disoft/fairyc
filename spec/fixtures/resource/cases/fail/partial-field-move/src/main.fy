module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

resource Pair {
    left: Token;
    right: Token;

    drop fn(self: &mut Self) {}
}

fn invalid() -> i32 {
    let pair: Pair = Pair {
        left: Token { value: 1 },
        right: Token { value: 2 },
    };
    let extracted: Token = pair.left;
    extracted.value
}
