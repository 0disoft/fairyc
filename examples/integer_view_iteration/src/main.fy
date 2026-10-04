module main;
import std::io as io;

/// Clips stored readings without copying the view or allocating a second array.
fn void clamp(ref mut i32[] values, i32 lower, i32 upper) {
    for (let ref mut i32 item in values) {
        if (*item < lower) { *item = lower; }
        if (*item > upper) { *item = upper; }
    }
}

@effects(console)
fn i32 main() {
    var i32[5] readings = [-7, 0, 42, 9, 2147483647];
    let ref mut i32[] values = ref mut readings;
    clamp(values, 0, 100);
    let ref i32[] output = ref readings;
    for (let i32 value in output) {
        switch (io::write_i32(value)) { case .ok(_): {} case .err(_): { return 2; } };
        switch (io::write("\n")) { case .ok(_): {} case .err(_): { return 2; } };
    }
    return 0;
}
