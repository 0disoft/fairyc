module main;

import std::alloc as alloc;

fn i32 use_arena(alloc::Arena arena) {
    var alloc::Arena mutable_arena = arena;
    var i32 result = 0;
    region scratch in ref mut mutable_arena {
        result = switch (scratch.place(42)) {
            case .ok(stored): { *stored }
            case .err(_): { 0 }
        };
    }
    return result;
}

fn i32 main() {
    return switch (alloc::create()) {
        case .ok(arena): {
            use_arena(arena)
        }
        case .err(_): { 1 }
    };
}
