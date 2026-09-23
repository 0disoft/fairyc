module main;
@link_name("fc_budget_seed") unsafe extern C fn u32 seed_input();
@link_name("fc_budget_rounds") unsafe extern C fn u32 rounds_input();
@link_name("fc_budget_begin") unsafe extern C fn void begin();
@link_name("fc_budget_finish") unsafe extern C fn void finish(u32 value);

fn u32 accumulate(u32 seed, u32 rounds) {
    let u8[4] data = [7, 11, 13, 17];
    let ref u8[] view = ref data;
    var u32 value = seed;
    var u32 i = 0;
    while (i < rounds) {
        var usize index = 0;
        while (index < view.len) {
            value = (value * 33 + view[index] as u32 + i % 251) % 65521;
            index += 1;
        }
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
