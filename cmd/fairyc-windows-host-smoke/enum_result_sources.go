package main

const enumResultSource = `module enum_result;

struct Failure {
    i32 status;
}

enum Kind {
    Unknown;
    Value { i32 amount; };
}

fn Result<Kind, Failure> observe(bool accept) {
    return if (accept) {
        .ok(.Value { .amount = 20 })
    } else {
        .err(Failure { .status = 22 })
    };
}

fn i32 main() {
    let i32 success = switch (observe(true)) {
        case .ok(kind) : { switch (kind) {
            case .Unknown : { 0 }
            case .Value { .amount = amount } : { amount }
        } }
        case .err(_) : { 0 }
    };
    let i32 failure = switch (observe(false)) {
        case .ok(_) : { 0 }
        case .err(error) : { error.status }
    };
    return success + failure;
}
`

const enumResultPropagationSource = `module enum_result_propagation;

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
`
