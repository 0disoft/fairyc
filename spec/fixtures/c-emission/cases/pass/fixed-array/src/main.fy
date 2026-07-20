module main;

fn pick(values: [u8; 3], index: usize) -> u8 {
    values[index]
}

fn main() -> i32 {
    let values: [u8; 3] = [40, 41, 42];
    let flags: [bool; 2] = [false, true];
    let signed: [i32; 1] = [42];
    let wide: [u32; 1] = [42];
    let offsets: [usize; 1] = [2];
    if flags[1] && signed[0] == 42 && wide[0] == 42 && offsets[0] == 2 {
        pick(values, offsets[0]) as i32
    } else {
        1
    }
}

fn bounds_trap() -> i32 {
    let values: [i32; 2] = [40, 42];
    values[2]
}
