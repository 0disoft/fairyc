module main;

struct Snapshot {
    Option<i32> value;
    Result<i32, i32> adjustment;
}

enum Envelope {
    Missing;
    Present { Snapshot snapshot; Option<i32> fallback; };
}

fn i32 score(Envelope envelope) {
    return switch (envelope) {
        case .Missing : { 0 }
        case .Present { .snapshot = snapshot, .fallback = fallback } : {
            let i32 base = switch (snapshot.value) {
                case .some(value) : { value }
                case .none : { switch (fallback) {
                    case .some(value) : { value }
                    case .none : { 0 }
                } }
            };
            switch (snapshot.adjustment) {
                case .ok(value) : { base + value }
                case .err(error) : { base - error }
            }
        }
    };
}

fn i32 main() {
    let Snapshot snapshot = Snapshot {
        .value = .some(40),
        .adjustment = .ok(2),
    };
    return score(.Present { .snapshot = snapshot, .fallback = .none });
}
