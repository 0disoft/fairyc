module main;

fn T answer<T>() { return 42; }

fn T forward<T>() { return { (answer()) }; }

fn Option<T> pack<T>(T value) { return .some(value); }

fn Result<i32,u32> finish(bool success) {
    return if (success) { .ok(answer()) } else { .err(answer()) };
}

fn i32 main() {
    let Option<u32> packed = { pack(42) };
    let u32 right = switch (packed) {
        case .none: { forward() }
        case .some(value): { value }
    };
    let u32 same = answer<u32>();
    let Option<u32> optional = .some(answer());
    let Result<i32,u32> outcome = finish(right == same);
    return switch (outcome) {
        case .ok(actual): { actual }
        case .err(error): {
            switch (optional) {
                case .none: { 0 }
                case .some(expected): { if (expected == error) { 1 } else { 2 } }
            }
        }
    };
}
