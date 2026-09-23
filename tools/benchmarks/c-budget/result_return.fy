module main;
@link_name("fc_budget_seed") unsafe extern C fn u32 seed_input();
@link_name("fc_budget_rounds") unsafe extern C fn u32 rounds_input();
@link_name("fc_budget_begin") unsafe extern C fn void begin();
@link_name("fc_budget_finish") unsafe extern C fn void finish(u32 value);

fn Result<u32, u32> classify(u32 byte) {
    if (byte % 8 == 0) { return .err(0); }
    return .ok(byte);
}

fn u32 decode(u32 byte) {
    let u32 value = switch (classify(byte)) {
        case .ok(value) : { value }
        case .err(_) : { return 3; }
    };
    return value + 1;
}

fn u32 accumulate(u32 seed, u32 rounds) {
    var u32 value = 0;
    var u32 i = 0;
    while (i < rounds) {
        let u32 byte = (i + seed) % 128;
        value = (value + decode(byte)) % 65521;
        i += 1;
    }
    return value;
}

fn i32 main() {
    let u32 seed = unsafe { seed_input() };
    let u32 rounds = unsafe { rounds_input() };
    assert(seed < 65521 && rounds <= 50000000);
    unsafe { begin(); }
    let u32 result = accumulate(seed, rounds);
    unsafe { finish(result); }
    return 0;
}
