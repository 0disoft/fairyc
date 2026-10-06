module main;

enum Reading {
    Missing;
    Value { i64 amount; bool valid; };
}

fn Reading read() {
    return .Value { .valid = true, .amount = 4000000000 };
}

fn i32 main() {
    let i64 result = switch(read()) {
        case .Missing: { 0 }
        case .Value { .amount = amount, .valid = valid }: {
            if (valid) { amount } else { 0 }
        }
    };
    assert(result == 4000000000);
    return 0;
}
