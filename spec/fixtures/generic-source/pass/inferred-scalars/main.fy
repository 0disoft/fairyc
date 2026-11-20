module main;

fn T first<T>(T left, T right) { return left; }
fn T wrap<T>(T value) { return first(value, value); }

fn i32 main() {
    let u32 anchor = 40;
    let selected = first(anchor, 1);
    if (selected != anchor) { return 1; }
    let negative = first(-2, -1);
    if (negative != -2) { return 1; }
    let i32 tail = wrap(first(2, 0));
    return 40 + first<i32>(tail, 0);
}
