module main;
@link_name("fc_budget_seed") unsafe extern C fn u32 seed_input();
@link_name("fc_budget_rounds") unsafe extern C fn u32 rounds_input();
@link_name("fc_budget_begin") unsafe extern C fn void begin();
@link_name("fc_budget_finish") unsafe extern C fn void finish(u32 value);

fn void step(ref mut u32 slot, u32 i) {
    *slot = (*slot * 33 + i % 251) % 65521;
}

fn u32 checksum_through_ref(u32 seed, u32 rounds) {
    var u32 value = seed;
    var u32 i = 0;
    while (i < rounds) {
        step(ref mut value, i);
        i += 1;
    }
    return value;
}

fn i32 main() {
    let u32 seed = unsafe { seed_input() };
    let u32 rounds = unsafe { rounds_input() };
    assert(seed < 65521 && rounds <= 50000000);
    unsafe { begin(); }
    let u32 result = checksum_through_ref(seed, rounds);
    unsafe { finish(result); }
    return 0;
}
