module main;

fn i32 conflict() {
    var i32 value = 1;
    let ref i32 view = ref value;
    value = 2;
    return *view;
}
