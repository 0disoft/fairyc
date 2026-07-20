module main;

resource Handle {
    raw: *mut void;
    state: u32;

    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    let handle: Handle = Handle { raw: null, state: 42 };
    if handle.raw == null {
        if handle.state == 42 { 42 } else { 1 }
    } else {
        2
    }
}
