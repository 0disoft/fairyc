module main;

@effects(filesystem)
fn load() -> i32 { 42 }

fn main() -> i32 { load() }
