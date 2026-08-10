package main

const nestedCompactAggregateSource = `module nested_compact_aggregate;

struct Snapshot {
    Option<Option<i32>> value;
    Result<Option<i32>, Result<i32, i32>> adjustment;
}

enum Envelope {
    Missing;
    Present { Snapshot snapshot; Option<Result<i32, i32>> fallback; };
}

fn i32 score(Envelope envelope) {
    return switch (envelope) {
        case .Missing : { 0 }
        case .Present { .snapshot = snapshot, .fallback = fallback } : {
            let i32 base = switch (snapshot.value) {
                case .some(option) : { switch (option) {
                    case .some(value) : { value }
                    case .none : { 0 }
                } }
                case .none : { 0 }
            };
            let i32 adjustment = switch (snapshot.adjustment) {
                case .ok(option) : { switch (option) {
                    case .some(value) : { value }
                    case .none : { 0 }
                } }
                case .err(result) : { switch (result) {
                    case .ok(value) : { value }
                    case .err(error) : { -error }
                } }
            };
            let i32 fallback_value = switch (fallback) {
                case .some(result) : { switch (result) {
                    case .ok(value) : { value }
                    case .err(error) : { -error }
                } }
                case .none : { 0 }
            };
            base + adjustment + fallback_value
        }
    };
}

fn i32 main() {
    let Snapshot snapshot = Snapshot {
        .value = .some(.some(40)),
        .adjustment = .err(.ok(2)),
    };
    return score(.Present { .snapshot = snapshot, .fallback = .none });
}
`
