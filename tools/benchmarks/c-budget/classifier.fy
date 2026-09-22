module main;
@link_name("fc_budget_seed") unsafe extern C fn u32 seed_input();
@link_name("fc_budget_rounds") unsafe extern C fn u32 rounds_input();
@link_name("fc_budget_begin") unsafe extern C fn void begin();
@link_name("fc_budget_finish") unsafe extern C fn void finish(u32 value);

fn u32 classify(u32 seed, u32 rounds) {
    var u32 value = 0;
    var u32 i = 0;
    while (i < rounds) {
        let u32 byte = (i + seed) % 128;
        var u32 kind = 4;
        if (byte == 32 || byte == 10) { kind = 1; }
        else if (byte >= 48 && byte <= 57) { kind = 2; }
        else if (byte >= 65 && byte <= 90) { kind = 3; }
        value = (value + kind) % 65521;
        i += 1;
    }
    return value;
}

fn i32 main() {
    let u32 seed = unsafe { seed_input() };
    let u32 rounds = unsafe { rounds_input() };
    assert(seed < 65521 && rounds <= 50000000);
    unsafe { begin(); }
    let u32 result = classify(.seed = seed, .rounds = rounds);
    unsafe { finish(result); }
    return 0;
}
