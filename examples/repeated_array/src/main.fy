module main;
const i32 COUNT = 2 + 2;
fn i32 last(i32[4] values) { return values[3]; }
fn i32 main() {
    var i32 calls = 0;
    let i32[4] values = [{ calls += 1; calls }; COUNT];
    assert(calls == 1);
    assert(values[0] == 1);
    assert(values[3] == 1);
    assert(last(values) == 1);
    let bool[1] flags = [true; 1];
    let u32[2] unsigned = [4294967295; 2];
    let i64[2] wide = [2147483648; 2];
    let usize[2] sizes = [8; 0b10];
    let inferred = [-7; 0x2];
    assert(flags[0]);
    assert(unsigned[1] == 4294967295);
    assert(wide[1] == 2147483648);
    assert(sizes[1] == 8);
    assert(inferred[1] == -7);
    var u8[4096] buffer = [0; 4_096];
    assert(buffer.len == 4096);
    assert(buffer[4095] == 0);
    let ref mut u8[] view = ref mut buffer;
    view[2048] = 7;
    assert(buffer[0] == 0);
    assert(buffer[2048] == 7);
    assert(buffer[4095] == 0);
    var i32 divisor = 2;
    let i32[3] quotients = [8 / divisor; 3];
    assert(quotients[2] == 4);
    return 0;
}
