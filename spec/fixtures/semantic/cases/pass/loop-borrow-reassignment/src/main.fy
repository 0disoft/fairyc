module main;

fn select(run: bool, right: &i32) -> &i32 {
    var current: &i32 = right;
    var first: bool = true;
    while run {
        current = right;
        if first {
            first = false;
            continue;
        } else {
            break;
        };
    }
    current
}
