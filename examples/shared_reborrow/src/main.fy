module main;

struct State { i32 number; }
fn i32 State::read(ref State this) { return this.number; }

fn i32 read(ref i32 value) { return *value; }
fn ref i32 retain(ref i32 value) { return value; }
fn ref i32 forward(ref mut i32 value) { return retain(value); }
fn Option<ref i32> optional(ref i32 value) { return .some(value); }
fn void edit(ref mut i32 value) { *value = *value + 1; }
fn u8 first(ref u8[] view) { return view[0]; }
fn void fill(ref mut u8[] view) { view[0] = 9; }

fn i32 main() {
    var i32 number = 41;
    let ref mut i32 view = ref mut number;
    assert(read(view) == 41);
    edit(view);
    let ref i32 child = forward(view);
    assert(*child == 42);
    edit(view);
    let Option<ref i32> tagged = optional(view);
    switch (tagged) { case .some(value): { assert(*value == 43); } case .none: { assert(false); } };
    edit(view);
    assert(read(view) == 44);
    var u8[2] data = [3, 4];
    let ref mut u8[] buffer = ref mut data;
    assert(first(buffer) == 3);
    fill(buffer);
    assert(first(buffer) == 9);
    var State state = State { .number = 7 };
    let ref mut State receiver = ref mut state;
    assert(receiver.read() == 7);
    receiver.number = 8;
    assert(receiver.read() == 8);
    return 0;
}
