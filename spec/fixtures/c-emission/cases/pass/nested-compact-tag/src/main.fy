module main;

fn i32 score_option_result(Option<Result<i32, i32>> value) {
    return switch (value) {
        case .some(result) : { switch (result) {
            case .ok(inner) : { inner }
            case .err(error) : { -error }
        } }
        case .none : { 0 }
    };
}

fn i32 score_result_nested(Result<Option<i32>, Result<i32, i32>> value) {
    return switch (value) {
        case .ok(option) : { switch (option) {
            case .some(inner) : { inner }
            case .none : { 0 }
        } }
        case .err(result) : { switch (result) {
            case .ok(inner) : { inner }
            case .err(error) : { -error }
        } }
    };
}

fn i32 main() {
    let Option<Result<i32, i32>> first = .some(.ok(40));
    let Result<Option<i32>, Result<i32, i32>> second = .err(.ok(2));
    return score_option_result(first) + score_result_nested(second);
}
