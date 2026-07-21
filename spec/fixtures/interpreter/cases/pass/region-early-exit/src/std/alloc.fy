module std::alloc;

pub resource Arena {
    i32 opaque;

    cleanup {}
}

resource Token {
    i32 value;

    cleanup {}
}

fn i32 return_from_region() {
    var Arena arena = Arena { .opaque = 0 };
    let Token outside = Token { .value = 20 };
    region scratch in ref mut arena {
        let Token inner = Token { .value = 22 };
        let ref i32 stored = scratch.place(1);
        return outside.value + inner.value;
    }
}

fn Result<i32, i32> propagate_from_region(Result<i32, i32> input) {
    var Arena arena = Arena { .opaque = 0 };
    let Token outside = Token { .value = 10 };
    region scratch in ref mut arena {
        let Token inner = Token { .value = 32 };
        let ref i32 stored = scratch.place(2);
        let i32 value = input?;
        return .ok(value + outside.value + inner.value);
    }
}

fn i32 propagate_main() {
    return switch (propagate_from_region(.err(42))) {
        case .ok(value) : { -value }
        case .err(error) : { error }
    };
}

fn i32 nested_return() {
    var Arena outer_arena = Arena { .opaque = 0 };
    var Arena inner_arena = Arena { .opaque = 0 };
    let Token function_token = Token { .value = 2 };
    region outer in ref mut outer_arena {
        let Token outer_token = Token { .value = 10 };
        let ref i32 outer_stored = outer.place(3);
        region inner in ref mut inner_arena {
            let Token inner_token = Token { .value = 30 };
            let ref i32 inner_stored = inner.place(4);
            return function_token.value + outer_token.value + inner_token.value;
        }
    }
}

fn i32 conditional_region(bool flag) {
    var Arena arena = Arena { .opaque = 0 };
    let Token outside = Token { .value = 40 };
    region scratch in ref mut arena {
        let Token inner = Token { .value = 2 };
        if (flag) {
            return outside.value + inner.value;
        };
        let ref i32 stored = scratch.place(5);
    }
    return outside.value;
}

fn i32 conditional_fallthrough_main() {
    return conditional_region(false);
}
