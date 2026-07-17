module main;

import std::alloc as alloc;

fn invalid(arena: &mut alloc::Arena, value: i32) -> &i32 {
    region scratch in arena {
        let stored: &i32 = scratch.place(value);
        return stored;
    }
}
