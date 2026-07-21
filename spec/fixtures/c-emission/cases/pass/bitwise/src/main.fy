module main;

fn i32 main() {
    let i32 all = ~0;
    let i32 low = all & 63;
    let i32 mixed = low ^ 21;
    return mixed | 0;
}
