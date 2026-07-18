module std::alloc;

pub resource Arena {
    opaque: i32;

    drop fn(self: &mut Self) {}
}

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn break_from_region() -> i32 {
    var arena: Arena = Arena { opaque: 0 };
    let outside: Token = Token { value: 40 };
    var count: i32 = 0;
    while count < 3 {
        let body: Token = Token { value: 1 };
        region scratch in &mut arena {
            let inner: Token = Token { value: 1 };
            let stored: &i32 = scratch.place(count);
            count = count + 1;
            if count == 1 {
                break;
            };
        }
    }
    outside.value + count
}

fn continue_from_region() -> i32 {
    var arena: Arena = Arena { opaque: 0 };
    let outside: Token = Token { value: 40 };
    var count: i32 = 0;
    while count < 2 {
        let body: Token = Token { value: 1 };
        region scratch in &mut arena {
            let inner: Token = Token { value: 1 };
            let stored: &i32 = scratch.place(count);
            count = count + 1;
            continue;
        }
    }
    outside.value + count
}

fn nested_loop_targets_nearest() -> i32 {
    var outer: i32 = 0;
    while outer < 2 {
        var inner: i32 = 0;
        while inner < 3 {
            inner = inner + 1;
            if inner == 1 {
                continue;
            };
            break;
        }
        outer = outer + 1;
    }
    outer + 40
}
