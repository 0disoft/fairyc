module main;

fn invalid() -> &i32 {
    let value: i32 = 1;
    &value
}

fn invalid_parameter(input: i32) -> &i32 {
    &input
}
