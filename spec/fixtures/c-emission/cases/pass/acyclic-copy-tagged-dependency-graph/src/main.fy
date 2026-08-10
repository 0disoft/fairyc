module main;

struct Inner { i32 value; }

enum Failure {
    Missing;
    Code { i32 value; };
}

struct Packet {
    Option<Result<Inner, Failure>> primary;
    Result<Option<Failure>, i32> fallback;
}

enum Envelope {
    Missing;
    Present { Packet packet; Result<Option<Inner>, Failure> alternate; };
}

fn i32 failure_value(Failure failure) {
    return switch (failure) {
        case .Missing : { 0 }
        case .Code { .value = value } : { value }
    };
}

fn i32 score(Envelope envelope) {
    return switch (envelope) {
        case .Missing : { 0 }
        case .Present { .packet = packet, .alternate = alternate } : {
            let i32 primary = switch (packet.primary) {
                case .some(result) : { switch (result) {
                    case .ok(inner) : { inner.value }
                    case .err(failure) : { failure_value(failure) }
                } }
                case .none : { 0 }
            };
            let i32 fallback = switch (packet.fallback) {
                case .ok(option) : { switch (option) {
                    case .some(failure) : { failure_value(failure) }
                    case .none : { 0 }
                } }
                case .err(error) : { error }
            };
            let i32 alternate_value = switch (alternate) {
                case .ok(option) : { switch (option) {
                    case .some(inner) : { inner.value }
                    case .none : { 0 }
                } }
                case .err(failure) : { failure_value(failure) }
            };
            primary + fallback + alternate_value
        }
    };
}

fn i32 main() {
    let Packet packet = Packet {
        .primary = .some(.ok(Inner { .value = 40 })),
        .fallback = .err(2),
    };
    return score(.Present { .packet = packet, .alternate = .err(.Missing) });
}
