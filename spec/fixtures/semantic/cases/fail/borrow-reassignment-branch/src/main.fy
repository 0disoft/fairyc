module main;

fn choose(flag: bool, left: &i32, right: &i32) -> &i32 {
    var current: &i32 = left;
    if flag { current = left; 0 } else { current = right; 0 };
    current
}
