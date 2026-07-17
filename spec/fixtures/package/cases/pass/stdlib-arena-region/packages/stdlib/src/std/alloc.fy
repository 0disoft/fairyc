module std::alloc;

pub resource Arena {
    opaque: i32;

    drop fn(self: &mut Self) {}
}
