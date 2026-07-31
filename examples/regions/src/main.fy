module main;

import std::alloc as alloc;

struct Pair {
    i32 value;
}

fn i32 use_arena(alloc::Arena arena) {
    var alloc::Arena mutable_arena = arena;
    var i32 scalar_result = 0;
    var i32 array_result = 0;
    var i32 struct_result = 0;
    region scratch in ref mut mutable_arena {
        scalar_result = switch (scratch.place(42)) {
            case .ok(stored): { *stored }
            case .err(_): { 0 }
        };
        let u8[3] values = [40, 41, 42];
        array_result = switch (scratch.place(values)) {
            case .ok(stored): {
                let u8[3] placed = *stored;
                placed[2] as i32
            }
            case .err(_): { 0 }
        };
        struct_result = switch (scratch.place(Pair { .value = 42 })) {
            case .ok(stored): { (*stored).value }
            case .err(_): { 0 }
        };
    }
    return if (scalar_result == 42 && array_result == 42 && struct_result == 42) { 42 } else { 1 };
}

fn i32 main() {
    return switch (alloc::create()) {
        case .ok(arena): {
            use_arena(arena)
        }
        case .err(_): { 1 }
    };
}
