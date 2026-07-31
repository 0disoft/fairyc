module std::alloc;

pub resource Arena {
    i32 opaque;

    cleanup {}
}

pub struct AllocError {
    u32 status;
}

resource Token {
    i32 value;

    cleanup {}
}

fn i32 consume(Token token) {
    return token.value;
}

fn i32 main() {
    var Arena arena = Arena { .opaque = 0 };
    var Token kept = Token { .value = 1 };
    let i32 consumed = consume(kept);
    region scratch in ref mut arena {
        kept = Token { .value = 9 };
        let Token inner = Token { .value = 7 };
        discard scratch.place(42);
    }
    return kept.value;
}
