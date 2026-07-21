module main;

import std::alloc as alloc;

fn ref i32 invalid(ref mut alloc::Arena arena, i32 value) {
    region scratch in arena {
        let ref i32 stored = scratch.place(value);
        return stored;
    }
}
