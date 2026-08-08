module main;

@repr(C)
struct fairyc_safe_record {
    i32 _value;
}

fn i32 main() {
    let fairyc_safe_record record = fairyc_safe_record { ._value = 42 };
    return record._value;
}
