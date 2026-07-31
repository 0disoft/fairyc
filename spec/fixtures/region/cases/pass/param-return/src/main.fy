module main;

import std::alloc as alloc;

fn Result<ref i32, alloc::AllocError> store(Region scratch, i32 value) {
    return scratch.place(value);
}
