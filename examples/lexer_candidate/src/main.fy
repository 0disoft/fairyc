module main;
import scanner as scanner;

@effects(console)
fn i32 main() {
    return scanner::report(b"module main; // first candidate\nfn void main() {}\n");
}
