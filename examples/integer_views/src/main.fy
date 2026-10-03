module main;
fn void read_i32(ref i32[] values) { assert(values.len == 3 && values[0] == -2147483648); }
fn ref mut i32[] tail_i32(ref mut i32[] values) { return values[1..]; }
fn Option<ref mut i32[]> tagged_i32(ref mut i32[] values) { return .some(values[1..2]); }
fn void check_i32() {
    var i32[3] data = [-2147483648, 7, 2147483647];
    let ref i32[] initial = ref data;
    assert(initial[0] == -2147483648 && initial[2] == 2147483647);
    let ref mut i32[] view = ref mut data;
    read_i32(view);
    let ref mut i32[] child = tail_i32(view);
    child[0] += 2;
    assert(view[1] == 9);
    let Option<ref mut i32[]> tagged = tagged_i32(view);
    switch (tagged) { case .some(v): { v[0] = 11; } case .none: { assert(false); } };
    assert(view[1] == 11);
    let ref mut i32[] empty = view[3..];
    assert(empty.is_empty && empty.len == 0);
    var usize order = 0;
    let ref mut i32[] middle = view[{ order = order * 10 + 1; 1 }..{ order = order * 10 + 2; 2 }];
    assert(order == 12);
    middle[0] = 13;
    assert(data[1] == 13 && data[0] == -2147483648 && data[2] == 2147483647);
}

fn void read_u32(ref u32[] values) { assert(values.len == 3 && values[0] == 0); }
fn ref mut u32[] tail_u32(ref mut u32[] values) { return values[1..]; }
fn Option<ref mut u32[]> tagged_u32(ref mut u32[] values) { return .some(values[1..2]); }
fn void check_u32() {
    var u32[3] data = [0, 7, 4294967295];
    let ref u32[] initial = ref data;
    assert(initial[0] == 0 && initial[2] == 4294967295);
    let ref mut u32[] view = ref mut data;
    read_u32(view);
    let ref mut u32[] child = tail_u32(view);
    child[0] += 2;
    assert(view[1] == 9);
    let Option<ref mut u32[]> tagged = tagged_u32(view);
    switch (tagged) { case .some(v): { v[0] = 11; } case .none: { assert(false); } };
    assert(view[1] == 11);
    let ref mut u32[] empty = view[3..];
    assert(empty.is_empty && empty.len == 0);
    var usize order = 0;
    let ref mut u32[] middle = view[{ order = order * 10 + 1; 1 }..{ order = order * 10 + 2; 2 }];
    assert(order == 12);
    middle[0] = 13;
    assert(data[1] == 13 && data[0] == 0 && data[2] == 4294967295);
}

fn void read_usize(ref usize[] values) { assert(values.len == 3 && values[0] == 0); }
fn ref mut usize[] tail_usize(ref mut usize[] values) { return values[1..]; }
fn Option<ref mut usize[]> tagged_usize(ref mut usize[] values) { return .some(values[1..2]); }
fn void check_usize() {
    var usize[3] data = [0, 7, 18446744073709551615];
    let ref usize[] initial = ref data;
    assert(initial[0] == 0 && initial[2] == 18446744073709551615);
    let ref mut usize[] view = ref mut data;
    read_usize(view);
    let ref mut usize[] child = tail_usize(view);
    child[0] += 2;
    assert(view[1] == 9);
    let Option<ref mut usize[]> tagged = tagged_usize(view);
    switch (tagged) { case .some(v): { v[0] = 11; } case .none: { assert(false); } };
    assert(view[1] == 11);
    let ref mut usize[] empty = view[3..];
    assert(empty.is_empty && empty.len == 0);
    var usize order = 0;
    let ref mut usize[] middle = view[{ order = order * 10 + 1; 1 }..{ order = order * 10 + 2; 2 }];
    assert(order == 12);
    middle[0] = 13;
    assert(data[1] == 13 && data[0] == 0 && data[2] == 18446744073709551615);
}

fn i32 main() { check_i32(); check_u32(); check_usize(); return 0; }
