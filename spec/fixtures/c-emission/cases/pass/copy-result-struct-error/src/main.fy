module main;

struct ConsoleError {
    u32 status;
    usize written;
}

fn Result<usize, ConsoleError> write(bool accept) {
    return if (accept) {
        .ok(42)
    } else {
        .err(ConsoleError { .status = 3, .written = 7 })
    };
}

fn i32 main() {
    return switch (write(false)) {
        case .ok(_) : { 0 }
        case .err(error) : { if (error.status == 3) { 42 } else { 0 } }
    };
}
