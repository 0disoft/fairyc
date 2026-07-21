module main;

resource Handle {
    ptr mut void raw;
    u32 state;

    cleanup {}
}

fn i32 main() {
    let Handle handle = Handle { .raw = null, .state = 42 };
    return if (handle.raw == null) {
        if (handle.state == 42) { 42 } else { 1 }
    } else {
        2
    };
}
