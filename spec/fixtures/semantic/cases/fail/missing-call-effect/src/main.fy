module main;

@effects(filesystem)
fn i32 load() { return 42; }

fn i32 main() { return load(); }
