module main;
fn ref mut u8[] tail(ref mut u8[] view) { return view[1..]; }
fn void fill(ref mut u8[] view) { view[0] = 9; }
fn void read(ref u8[] view) { assert(view.len > 0); }
fn i32 main() {
    var u8[4] data = [1,2,3,4];
    let ref mut u8[] view = ref mut data;
    let ref mut u8[] child = view[1..3];
    child[0] = 7;
    let ref mut u8[] leaf = child[..1];
    fill(leaf);
    assert(child[0] == 9);
    assert(view[1] == 9);
    let ref mut u8[] returned = tail(view);
    fill(returned);
    read(returned);
    let ref mut u8[] whole = view[..];
    let usize end = whole.len;
    let ref mut u8[] empty = whole[end..end];
    assert(empty.len == 0);
    whole[3] = 8;
    assert(view[3] == 8);
    var usize order = 0;
    let ref mut u8[] ordered = view[{ order = order * 10 + 1; 1 }..{ order = order * 10 + 2; 3 }];
    assert(order == 12);
    fill(ordered);
    assert(view[1] == 9);
    return 0;
}
