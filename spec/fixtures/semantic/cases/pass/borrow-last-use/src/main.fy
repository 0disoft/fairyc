module main;

fn read(value: &i32) -> i32 {
    *value
}

fn update() -> i32 {
    var value: i32 = 1;
    let view: &i32 = &value;
    let before: i32 = read(view);
    value = 2;
    before + value
}
