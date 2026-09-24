module main;
import std::vec as v;

@link_name("fc_budget_seed") unsafe extern C fn u32 seed_input();
@link_name("fc_budget_rounds") unsafe extern C fn u32 rounds_input();
@link_name("fc_budget_begin") unsafe extern C fn void begin();
@link_name("fc_budget_finish") unsafe extern C fn void finish(u32 value);

fn void checksum_through_vector(u32 seed, u32 rounds) {
    var v::Vec<u32> values = v::empty<u32>();
    assert(switch (v::push<u32>(ref mut values, 7)) { case .ok(n): { n == 1 } case .err(_): { false } });
    assert(switch (v::push<u32>(ref mut values, 11)) { case .ok(n): { n == 2 } case .err(_): { false } });
    assert(switch (v::push<u32>(ref mut values, 13)) { case .ok(n): { n == 3 } case .err(_): { false } });
    assert(switch (v::push<u32>(ref mut values, 17)) { case .ok(n): { n == 4 } case .err(_): { false } });
    var u32 value = seed;
    unsafe { begin(); }
    var u32 i = 0;
    while (i < rounds) {
        let usize bound = v::len<u32>(ref values);
        var usize index = 0;
        while (index < bound) {
            let ref u32 item = v::at<u32>(ref values, index);
            value = (value * 33 + *item + i % 251) % 65521;
            index += 1;
        }
        i += 1;
    }
    unsafe { finish(value); }
}

fn i32 main() {
    let u32 seed = unsafe { seed_input() };
    let u32 rounds = unsafe { rounds_input() };
    assert(seed < 65521 && rounds <= 50000000);
    checksum_through_vector(seed, rounds);
    return 0;
}
