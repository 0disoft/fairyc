module main;

enum Kind {
    Unknown;
    Value { i32 amount; };
}

fn Option<Kind> choose(bool present) {
    return if (present) {
        .some(.Value { .amount = 20 })
    } else {
        .none
    };
}

fn i32 score(Option<Kind> kind) {
    return switch (kind) {
        case .some(value) : { switch (value) {
            case .Unknown : { 0 }
            case .Value { .amount = amount } : { amount }
        } }
        case .none : { 22 }
    };
}

fn i32 main() {
    return score(choose(true)) + score(choose(false));
}
