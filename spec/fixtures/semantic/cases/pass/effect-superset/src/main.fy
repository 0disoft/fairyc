module main;

@effects(filesystem)
fn load() -> i32 { 42 }

@effects(filesystem, console)
fn main() -> i32 { load() }
