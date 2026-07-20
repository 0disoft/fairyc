module main;

fn select(flag: bool, left: &i32, right: &i32) -> &i32 {
    var current: &i32 = left;
    if flag { current = right; 0 } else { current = right; 0 };
    current
}
