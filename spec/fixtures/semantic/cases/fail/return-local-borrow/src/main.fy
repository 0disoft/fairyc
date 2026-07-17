module main;

fn invalid() -> &i32 {
    let value: i32 = 1;
    &value
}
