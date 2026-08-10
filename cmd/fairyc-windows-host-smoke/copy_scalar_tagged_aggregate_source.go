package main

const copyScalarTaggedAggregateSource = `module copy_scalar_tagged_aggregate;

struct Snapshot {
    Option<u32> count;
    Result<u8, usize> adjustment;
}

enum Envelope {
    Missing;
    Present { Snapshot snapshot; Option<usize> fallback; };
}

fn i32 score(Envelope envelope) {
    return switch (envelope) {
        case .Missing : { 0 }
        case .Present { .snapshot = snapshot, .fallback = fallback } : {
            let bool count_ok = switch (snapshot.count) {
                case .some(value) : { value == 40 }
                case .none : { false }
            };
            let bool adjustment_ok = switch (snapshot.adjustment) {
                case .ok(value) : { value == 2 }
                case .err(error) : { error == 0 }
            };
            let bool fallback_ok = switch (fallback) {
                case .some(value) : { value == 0 }
                case .none : { true }
            };
            if (count_ok && adjustment_ok && fallback_ok) { 42 } else { 0 }
        }
    };
}

fn i32 main() {
    let Snapshot snapshot = Snapshot {
        .count = .some(40),
        .adjustment = .ok(2),
    };
    return score(.Present { .snapshot = snapshot, .fallback = .none });
}
`
