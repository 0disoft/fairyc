module main;

@const_eval
fn i32 square(i32 value) {
    return value * value;
}

@const_eval
fn u32 widen(u8 value) {
    return value as u32;
}

const i32 ANSWER = square(6) + 6;
const u8 BYTE = 40 + 2;
const u32 WIDE = widen(BYTE);
const bool READY = false || (ANSWER == 42 && WIDE == 42);
const i32 SELECTED = if (READY) { ANSWER } else { 1 / 0 };

static_assert(READY && SELECTED == 42);

fn i32 main() {
    return SELECTED;
}
