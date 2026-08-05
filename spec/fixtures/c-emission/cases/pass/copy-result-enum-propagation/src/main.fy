module main;

enum Failure {
    Denied;
    Code { i32 status; };
}

fn Result<i32, Failure> pass(Result<i32, Failure> input) {
    let i32 value = input?;
    return .ok(value + 1);
}

fn i32 score(Result<i32, Failure> result) {
    return switch (result) {
        case .ok(value) : { value }
        case .err(failure) : { switch (failure) {
            case .Denied : { 0 }
            case .Code { .status = status } : { status }
        } }
    };
}

fn i32 main() {
    return score(pass(.ok(19))) + score(pass(.err(.Code { .status = 22 })));
}
