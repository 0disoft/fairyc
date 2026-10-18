package main

const taggedArraySource = `module tagged_array;

fn Option<u8[3]> choose(bool present) {
    return if (present) { .some([10, 11, 12]) } else { .none };
}

fn Result<u8[3], u8[2]> forward(Result<u8[3], u8[2]> input) {
    let u8[3] values = input?;
    return .ok(values);
}

fn i32 score_option(Option<u8[3]> value) {
    return switch (value) {
        case .some(values) : { values[0] as i32 }
        case .none : { 2 }
    };
}

fn i32 score_result(Result<u8[3], u8[2]> value) {
    return switch (value) {
        case .ok(values) : { values[0] as i32 }
        case .err(error) : { error[0] as i32 }
    };
}

fn i32 main() {
    return score_option(choose(true)) + score_option(choose(false)) +
        score_result(forward(.ok([10, 0, 0]))) +
        score_result(forward(.err([20, 0])));
}
`
