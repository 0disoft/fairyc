module main;

@effects(filesystem)
fn i32 load() { return 42; }

@effects(filesystem, console)
fn i32 main() { return load(); }
