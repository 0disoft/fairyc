module std::alloc;

pub resource Arena {
    i32 opaque;

    cleanup {}
}

pub struct AllocError {
    u32 status;
}
