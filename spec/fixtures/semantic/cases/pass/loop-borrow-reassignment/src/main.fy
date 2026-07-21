module main;

fn ref i32 select(bool run, ref i32 right) {
    var ref i32 current = right;
    var bool first = true;
    while (run) {
        current = right;
        if (first) {
            first = false;
            continue;
        } else {
            break;
        };
    }
    return current;
}
