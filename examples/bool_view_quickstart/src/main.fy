module main;
import flags as flags;
import std::io as io;

@effects(console)
fn i32 main() {
    var bool[8] occupied = [false; 8];
    let ref mut bool[] slots = ref mut occupied;
    // Mark only the requested range, without copying the backing array.
    flags::fill(slots[2..5], true);
    let usize used = flags::count_set(slots);
    let usize start = 2;
    var usize next = occupied.len;
    switch (flags::first_clear(slots[start..])) {
        case .some(index): { next = index + start; }
        case .none: { return 1; }
    };
    assert(used == 3 && next == 5);
    slots[next] = true;
    assert(flags::count_set(slots) == 4);
    if (!io::write("occupied=").is_ok || !io::write_usize(used).is_ok
        || !io::write(" next=").is_ok || !io::write_usize(next).is_ok
        || !io::write(" after=").is_ok || !io::write_usize(flags::count_set(slots)).is_ok
        || !io::write("\n").is_ok) { return 2; }
    return 0;
}
