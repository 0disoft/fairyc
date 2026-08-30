module api;

pub struct Pair {
    i32 left;
    u32 right;
}

pub enum Status {
    ready;
    failed { i32 code; };
}

@abilities(store, key)
pub resource Handle {
    i32 state;
    cleanup { let i32 observed = this.state; }
}

@link_name("raw_v1")
pub unsafe extern C fn i32 raw(i32 value);

pub fn i32 read(Pair pair) { return pair.left; }
fn i32 private_helper() { return 1; }
