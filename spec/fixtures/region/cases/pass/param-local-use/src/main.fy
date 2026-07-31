module main;

fn i32 read(Region scratch, i32 value) {
    return switch (scratch.place(value)) {
        case .ok(stored): { *stored }
        case .err(_): { 0 }
    };
}
