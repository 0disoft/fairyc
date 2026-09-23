module main;
import std::vec as v;

@link_name("fc_budget_seed") unsafe extern C fn u32 seed_input();
@link_name("fc_budget_rounds") unsafe extern C fn u32 rounds_input();
@link_name("fc_budget_begin") unsafe extern C fn void begin();
@link_name("fc_budget_finish") unsafe extern C fn void finish(u32 value);

fn void step(ref mut v::Vec<u32> values, u32 i) {
    let ref mut u32 slot = v::at_mut<u32>(values, 0);
    *slot = (*slot * 33 + i % 251) % 65521;
}

fn void checksum_through_vector(u32 seed, u32 rounds) {
    var v::Vec<u32> values = v::empty<u32>();
    assert(switch (v::push<u32>(ref mut values, seed)) {
        case .ok(n) : { n == 1 }
        case .err(_) : { false }
    });
    unsafe { begin(); }
    var u32 i = 0;
    while (i < rounds) {
        step(ref mut values, i);
        i += 1;
    }
    let ref u32 result = v::at<u32>(ref values, 0);
    unsafe { finish(*result); }
}

fn i32 main() {
    let u32 seed = unsafe { seed_input() };
    let u32 rounds = unsafe { rounds_input() };
    assert(seed < 65521 && rounds <= 50000000);
    checksum_through_vector(seed, rounds);
    return 0;
}
