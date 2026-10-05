module main;
struct Pair { i64 amount; bool ready; }
struct Box { Pair pair; }
fn Box identity(Box value) { return value; }
fn i64 main() {
    var box = Box { .pair = Pair { .ready = true, .amount = 4000000000 } };
    box.pair.amount += 2;
    let copy = identity(box);
    return copy.pair.amount;
}
