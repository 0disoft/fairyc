module main;

fn ref u8[] tail(ref u8[] bytes) { return bytes[1..]; }

fn i32 main() {
    var u8[4] packet = [0,1,2,255];
    let ref u8[] bytes = ref packet;
    let payload = tail(bytes);
    assert(payload.len == 3 && payload[2] == 255);
    assert(payload[..0].is_empty && bytes[4..].is_empty);
    packet[0] = 9;
    return if (packet[0] == 9) { 0 } else { 1 };
}
