package main

const aggregateTaggedSource = `module aggregate_tagged;

struct Inner { i32 value; }

enum Failure {
    Missing;
    Code { i32 value; };
}

struct Outer {
    Option<Inner> inner;
    Result<Inner, Failure> result;
}

enum Envelope {
    Missing;
    Present { Outer outer; Option<Failure> fallback; };
}

fn i32 score(Envelope envelope) {
    return switch (envelope) {
        case .Missing : { 0 }
        case .Present { .outer = outer, .fallback = fallback } : {
            let i32 base = switch (outer.inner) {
                case .some(inner) : { inner.value }
                case .none : { switch (fallback) {
                    case .some(failure) : { switch (failure) {
                        case .Missing : { 1 }
                        case .Code { .value = value } : { value }
                    } }
                    case .none : { 0 }
                } }
            };
            switch (outer.result) {
                case .ok(inner) : { base + inner.value }
                case .err(failure) : { switch (failure) {
                    case .Missing : { base }
                    case .Code { .value = value } : { base + value }
                } }
            }
        }
    };
}

fn i32 main() {
    let Outer outer = Outer {
        .inner = .some(Inner { .value = 40 }),
        .result = .err(.Code { .value = 2 }),
    };
    return score(.Present { .outer = outer, .fallback = .none });
}
`
