module main;

resource Success {
    i32 value;
    cleanup {}
}

resource Failure {
    i32 value;
    cleanup {}
}

fn i32 main() {
    let Result<Success, Failure> wrapped = .ok(Success { .value = 17 });
    discard wrapped;
    return 42;
}
