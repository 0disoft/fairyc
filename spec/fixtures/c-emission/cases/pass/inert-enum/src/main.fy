module main;

enum Message {
    Empty;
    Value { amount: i32; };
    Pair { left: i32; right: i32; };
}

fn score(message: Message) -> i32 {
    match message {
        Message::Empty => 0,
        Message::Value { amount: value } => value,
        Message::Pair { left: first, right: second } => first + second,
    }
}

fn main() -> i32 {
    score(Message::Pair { left: 20, right: 22 })
}
