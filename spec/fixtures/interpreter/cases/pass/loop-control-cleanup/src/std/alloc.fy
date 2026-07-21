module std::alloc;

pub resource Arena {
    i32 opaque;

    cleanup {}
}

resource Token {
    i32 value;

    cleanup {}
}

fn i32 break_from_region() {
    var Arena arena = Arena { .opaque = 0 };
    let Token outside = Token { .value = 40 };
    var i32 count = 0;
    while (count < 3) {
        let Token body = Token { .value = 1 };
        region scratch in ref mut arena {
            let Token inner = Token { .value = 1 };
            let ref i32 stored = scratch.place(count);
            count = count + 1;
            if (count == 1) {
                break;
            };
        }
    }
    return outside.value + count;
}

fn i32 continue_from_region() {
    var Arena arena = Arena { .opaque = 0 };
    let Token outside = Token { .value = 40 };
    var i32 count = 0;
    while (count < 2) {
        let Token body = Token { .value = 1 };
        region scratch in ref mut arena {
            let Token inner = Token { .value = 1 };
            let ref i32 stored = scratch.place(count);
            count = count + 1;
            continue;
        }
    }
    return outside.value + count;
}

fn i32 nested_loop_targets_nearest() {
    var i32 outer = 0;
    while (outer < 2) {
        var i32 inner = 0;
        while (inner < 3) {
            inner = inner + 1;
            if (inner == 1) {
                continue;
            };
            break;
        }
        outer = outer + 1;
    }
    return outer + 40;
}
