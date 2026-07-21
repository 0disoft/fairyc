module main;

enum Message {
    Empty;
    Value { i32 amount; };
    Pair { i32 left; i32 right; };
}

fn i32 score(Message message) {
    return switch (message) {
        case .Empty : { 0 }
        case .Value { .amount = value } : { value }
        case .Pair { .left = first, .right = second } : { first + second }
    };
}

fn i32 main() {
    return score(.Pair { .left = 20, .right = 22 });
}
