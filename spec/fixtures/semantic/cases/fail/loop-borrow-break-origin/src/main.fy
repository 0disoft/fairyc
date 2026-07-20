module main;

fn select(run: bool, left: &i32, right: &i32) -> &i32 {
    var current: &i32 = left;
    while run {
        current = right;
        break;
    }
    current
}
