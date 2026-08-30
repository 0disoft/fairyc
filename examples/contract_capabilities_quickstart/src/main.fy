module main;

@abilities(drop, store, key)
resource Receipt {
    i32 value;
    cleanup {}
}

@effects(storage_read)
fn i32 read_value() {
    return 42;
}

@effects(storage_read)
fn i32 main() {
    let Receipt receipt = Receipt { .value = 42 };
    return read_value();
}
