module main;

fn conflict() -> i32 {
    var value: i32 = 1;
    let view: &i32 = &value;
    value = 2;
    *view
}
