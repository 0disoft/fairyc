module main;

fn main() -> i32 {
    var count: i32 = 0;
    var total: i32 = 0;
    while count < 6 {
        total = total + 7;
        count = count + 1;
    }
    total
}
