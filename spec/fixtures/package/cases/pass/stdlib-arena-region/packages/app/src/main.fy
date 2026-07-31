module main;

import std::alloc as alloc;

fn Result<ref i32, alloc::AllocError> invalid(ref mut alloc::Arena arena, i32 value) {
    region scratch in arena {
        return scratch.place(value);
    }
}
