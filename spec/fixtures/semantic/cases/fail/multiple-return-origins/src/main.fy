module main;

fn choose(flag: bool, left: &i32, right: &i32) -> &i32 {
    if flag { left } else { right }
}
