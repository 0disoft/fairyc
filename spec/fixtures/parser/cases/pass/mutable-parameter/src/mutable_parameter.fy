module mutable_parameter;

fn i32 update(i32 input) {
    var i32 value = input;
    value = 42;
    return value;
}
