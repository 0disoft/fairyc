module pointer_provider;
@repr(C) pub struct Packet { i32 value; }
@repr(C) pub struct Other { i32 value; }
pub fn ptr Packet identity(ptr Packet value) { return value; }
