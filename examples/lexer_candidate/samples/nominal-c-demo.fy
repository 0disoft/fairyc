module main;
struct Unit { }
struct Counter { Unit marker; i64 value; bool ready; }
fn Counter make(i64 value) {
    return Counter { .marker = Unit { }, .value, .ready = true };
}
fn i32 main() {
    var counter = make(4000000000);
    let copied = counter;
    counter.value += 2;
    assert(copied.value == 4000000000);
    assert(counter.value == 4000000002 && counter.ready);
    return 0;
}
