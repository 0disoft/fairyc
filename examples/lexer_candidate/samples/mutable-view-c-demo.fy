module main;

fn ref mut u8[] payload(ref mut u8[] packet) { return packet[1..]; }

fn i32 main() {
    var u8[4] packet=[0,1,2,255];
    let ref mut u8[] bytes=ref mut packet;
    let data=payload(bytes);
    data[0]=8;
    data[1]+=3;
    assert(data[0]==8 && data[1]==5);
    bytes[0]=9;
    assert(packet[0]==9 && packet[1]==8 && packet[2]==5);
    return 0;
}
