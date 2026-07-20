module main;

fn update(bytes: &mut [u8]) -> u8 {
    bytes[1] = 42;
    bytes[1]
}

fn bounds_trap() -> i32 {
    var values: [u8; 3] = [1, 2, 3];
    values[3] = 42;
    0
}

fn main() -> i32 {
    var values: [u8; 3] = [1, 2, 3];
    values[0] = 40;
    let changed: u8 = update(&mut values);
    if values[0] == 40 && values[1] == changed && values[1] == 42 {
        42
    } else {
        1
    }
}
