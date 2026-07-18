module std::alloc;

pub resource Arena {
    opaque: i32;

    drop fn(self: &mut Self) {}
}

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn consume(token: Token) -> i32 {
    token.value
}

fn main() -> i32 {
    var arena: Arena = Arena { opaque: 0 };
    var kept: Token = Token { value: 1 };
    let consumed: i32 = consume(kept);
    region scratch in &mut arena {
        kept = Token { value: 9 };
        let inner: Token = Token { value: 7 };
        let stored: &i32 = scratch.place(42);
    }
    kept.value
}
