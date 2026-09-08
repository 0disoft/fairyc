module main;

import std::vec as vec;
import std::io as io;

pub fn i32 total() {
    var vec::Vec<i32> values = vec::empty<i32>();
    assert(switch (vec::push<i32>(ref mut values, 20)) {
        case .ok(length) : { length == 1 }
        case .err(_) : { false }
    });
    assert(switch (vec::push<i32>(ref mut values, 22)) {
        case .ok(length) : { length == 2 }
        case .err(_) : { false }
    });
    let i32 first = *vec::at<i32>(ref values, 0);
    let i32 last = switch (vec::pop<i32>(ref mut values)) {
        case .some(value) : { value }
        case .none : { 0 }
    };
    assert(vec::clear<i32>(ref mut values) == 1);
    assert(vec::is_empty<i32>(ref values));
    return first + last;
}

@effects(console)
fn i32 main() {
    return switch (io::write_i32(total())) {
        case .ok(_) : { 0 }
        case .err(_) : { 1 }
    };
}
