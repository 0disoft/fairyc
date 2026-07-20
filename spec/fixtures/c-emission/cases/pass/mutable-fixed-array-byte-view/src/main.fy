module main;

fn length(bytes: &mut [u8]) -> usize {
    bytes.len
}

fn main() -> i32 {
    var values: [u8; 3] = [40, 41, 42];
    if length(&mut values) == 3 {
        values[2] as i32
    } else {
        1
    }
}
