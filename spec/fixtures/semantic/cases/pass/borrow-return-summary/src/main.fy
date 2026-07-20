module main;

fn relay(left: &i32, right: &i32) -> &i32 {
    second(right, left)
}

fn second(left: &i32, right: &i32) -> &i32 {
    right
}
