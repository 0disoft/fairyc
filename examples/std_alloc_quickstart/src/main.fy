module main;

import std::alloc as alloc;

fn i32 use_arena(alloc::Arena arena) {
    return 42;
}

fn i32 main() {
    return switch (alloc::create()) {
        case .ok(arena): { use_arena(arena) }
        case .err(error): {
            if (alloc::error_status(error) == 0) { 1 } else { 2 }
        }
    };
}
