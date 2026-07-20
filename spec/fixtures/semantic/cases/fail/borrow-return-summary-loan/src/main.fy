module main;

fn second(left: &[u8], right: &[u8]) -> &[u8] {
    right
}

fn main() -> i32 {
    let left: [u8; 1] = [1];
    var right: [u8; 1] = [42];
    let selected: &[u8] = second(&left, &right);
    right = [0];
    selected[0] as i32
}
