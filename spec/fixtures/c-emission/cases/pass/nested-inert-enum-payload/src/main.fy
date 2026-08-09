module main;

enum Envelope {
    Missing;
    Present { State state; };
}

enum State {
    Empty;
    Code { i32 value; };
}

fn i32 score(Envelope envelope) {
    return switch (envelope) {
        case .Missing : { 0 }
        case .Present { .state = state } : { switch (state) {
            case .Empty : { 1 }
            case .Code { .value = value } : { value }
        } }
    };
}

fn i32 main() {
    return score(.Present { .state = .Code { .value = 42 } });
}
