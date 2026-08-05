module main;

struct Failure {
    i32 status;
}

enum Kind {
    Unknown;
    Value { i32 amount; };
}

fn Result<Kind, Failure> observe(bool accept) {
    return if (accept) {
        .ok(.Value { .amount = 20 })
    } else {
        .err(Failure { .status = 22 })
    };
}

fn i32 main() {
    let i32 success = switch (observe(true)) {
        case .ok(kind) : { switch (kind) {
            case .Unknown : { 0 }
            case .Value { .amount = amount } : { amount }
        } }
        case .err(_) : { 0 }
    };
    let i32 failure = switch (observe(false)) {
        case .ok(_) : { 0 }
        case .err(error) : { error.status }
    };
    return success + failure;
}
