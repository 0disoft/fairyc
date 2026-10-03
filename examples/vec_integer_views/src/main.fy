module main;
import std::vec as vec;

fn void append_i32(ref mut vec::Vec<i32> values, i32 value) {
    assert(switch (vec::push<i32>(values, value)) { case .ok(_): { true } case .err(_): { false } });
}
fn ref i32[] tail_i32(ref vec::Vec<i32> values) { return vec::view_i32(values)[1..]; }
fn Option<ref i32[]> shared_tagged_i32(ref vec::Vec<i32> values) { return .some(tail_i32(values)); }
fn ref mut i32[] middle_i32(ref mut vec::Vec<i32> values) { return vec::view_mut_i32(values)[1..3]; }
fn Option<ref mut i32[]> tagged_i32(ref mut vec::Vec<i32> values) { return .some(middle_i32(values)); }
fn void read_i32(ref i32[] values) { assert(values.len > 0); }
fn void reverse_i32(ref mut i32[] values) {
    var usize left = 0;
    var usize right = values.len;
    while (left < right) {
        right -= 1;
        let i32 saved = values[left];
        values[left] = values[right];
        values[right] = saved;
        left += 1;
    }
}
fn void check_i32() {
    var i32[3] array = [1,2,3];
    let ref i32[] fixed_shared = ref array;
    var i32 fixed_sum = 0;
    for (let i32 item in fixed_shared) { fixed_sum += item; }
    assert(fixed_sum == 6);
    let ref mut i32[] fixed_mut = ref mut array;
    var usize fixed_index = 0;
    for (let i32 item in fixed_mut) {
        if (fixed_index == 0) { fixed_mut[1] = 8; }
        if (fixed_index == 1) { assert(item == 8); }
        fixed_index += 1;
    }
    assert(array[1] == 8 && fixed_index == 3);
    var vec::Vec<i32> values = vec::empty<i32>();
    let ref i32[] empty = vec::view_i32(ref values);
    assert(empty.is_empty && empty[..].len == 0 && empty[0..0].is_empty);
    for (let i32 byte in empty) { assert(false); }
    let ref mut i32[] writable_empty = vec::view_mut_i32(ref mut values);
    assert(writable_empty.is_empty && writable_empty[..].len == 0);
    for (let i32 byte in writable_empty) { assert(false); }
    append_i32(ref mut values, 0);
    let ref mut i32[] one = vec::view_mut_i32(ref mut values);
    for (let i32 byte in one) { assert(byte == 0); one[0] = 1; }
    append_i32(ref mut values, 127);
    append_i32(ref mut values, 128);
    append_i32(ref mut values, 255);
    let ref i32[] shared = vec::view_i32(ref values);
    var i32 sum = 0;
    for (let i32 byte in shared) { sum += byte; }
    assert(sum == 511 && shared[0] == 1 && shared[3] == 255);
    let ref i32[] returned = tail_i32(ref values);
    assert(returned.len == 3 && returned[0] == 127);
    let Option<ref i32[]> shared_result = shared_tagged_i32(ref values);
    switch (shared_result) { case .some(view): { assert(view.len == 3 && view[0] == 127); } case .none: { assert(false); } };
    let ref mut i32[] writable = vec::view_mut_i32(ref mut values);
    let ref mut i32[] child = writable[1..3];
    reverse_i32(child);
    read_i32(child);
    assert(writable[1] == 128 && writable[2] == 127);
    let ref mut i32[] nested = writable[1..][..1];
    nested[0] += 1;
    assert(writable[1] == 129);
    let Option<ref mut i32[]> result = tagged_i32(ref mut values);
    switch (result) { case .some(view): { view[0] = 130; } case .none: { assert(false); } };
    let ref mut i32[] loop_view = vec::view_mut_i32(ref mut values);
    var usize index = 0;
    for (let i32 byte in loop_view) {
        if (index == 0) { loop_view[1] = 42; }
        if (index == 1) { assert(byte == 42); }
        index += 1;
    }
    assert(index == 4);
    let usize capacity = vec::capacity<i32>(ref values);
    assert(switch (vec::reserve<i32>(ref mut values, capacity + 16)) {
        case .ok(new_capacity): { new_capacity >= capacity + 16 }
        case .err(_): { false }
    });
    let ref mut i32[] grown = vec::view_mut_i32(ref mut values);
    grown[2] = 99;
    assert(grown[1] == 42 && grown[2] == 99 && grown[3] == 255);
    append_i32(ref mut values, 7);
    assert(switch (vec::pop<i32>(ref mut values)) { case .some(byte): { byte == 7 } case .none: { false } });
    assert(vec::truncate<i32>(ref mut values, 2) == 2);
    assert(vec::clear<i32>(ref mut values) == 2);
    assert(vec::view_i32(ref values).is_empty);
    values = vec::empty<i32>();
    append_i32(ref mut values, 55);
    let ref mut i32[] last = vec::view_mut_i32(ref mut values);
    last[0] = 56;
    let vec::Vec<i32> moved = values;
    assert(vec::view_i32(ref moved)[0] == 56);
    var vec::Vec<i32> extremes = vec::empty<i32>();
    append_i32(ref mut extremes, 2147483647);
    append_i32(ref mut extremes, -2147483648);
    assert(vec::view_i32(ref extremes)[1] == -2147483648);
    assert(vec::view_i32(ref extremes)[0] == 2147483647);

}

fn void append_u32(ref mut vec::Vec<u32> values, u32 value) {
    assert(switch (vec::push<u32>(values, value)) { case .ok(_): { true } case .err(_): { false } });
}
fn ref u32[] tail_u32(ref vec::Vec<u32> values) { return vec::view_u32(values)[1..]; }
fn Option<ref u32[]> shared_tagged_u32(ref vec::Vec<u32> values) { return .some(tail_u32(values)); }
fn ref mut u32[] middle_u32(ref mut vec::Vec<u32> values) { return vec::view_mut_u32(values)[1..3]; }
fn Option<ref mut u32[]> tagged_u32(ref mut vec::Vec<u32> values) { return .some(middle_u32(values)); }
fn void read_u32(ref u32[] values) { assert(values.len > 0); }
fn void reverse_u32(ref mut u32[] values) {
    var usize left = 0;
    var usize right = values.len;
    while (left < right) {
        right -= 1;
        let u32 saved = values[left];
        values[left] = values[right];
        values[right] = saved;
        left += 1;
    }
}
fn void check_u32() {
    var u32[3] array = [1,2,3];
    let ref u32[] fixed_shared = ref array;
    var u32 fixed_sum = 0;
    for (let u32 item in fixed_shared) { fixed_sum += item; }
    assert(fixed_sum == 6);
    let ref mut u32[] fixed_mut = ref mut array;
    var usize fixed_index = 0;
    for (let u32 item in fixed_mut) {
        if (fixed_index == 0) { fixed_mut[1] = 8; }
        if (fixed_index == 1) { assert(item == 8); }
        fixed_index += 1;
    }
    assert(array[1] == 8 && fixed_index == 3);
    var vec::Vec<u32> values = vec::empty<u32>();
    let ref u32[] empty = vec::view_u32(ref values);
    assert(empty.is_empty && empty[..].len == 0 && empty[0..0].is_empty);
    for (let u32 byte in empty) { assert(false); }
    let ref mut u32[] writable_empty = vec::view_mut_u32(ref mut values);
    assert(writable_empty.is_empty && writable_empty[..].len == 0);
    for (let u32 byte in writable_empty) { assert(false); }
    append_u32(ref mut values, 0);
    let ref mut u32[] one = vec::view_mut_u32(ref mut values);
    for (let u32 byte in one) { assert(byte == 0); one[0] = 1; }
    append_u32(ref mut values, 127);
    append_u32(ref mut values, 128);
    append_u32(ref mut values, 255);
    let ref u32[] shared = vec::view_u32(ref values);
    var u32 sum = 0;
    for (let u32 byte in shared) { sum += byte; }
    assert(sum == 511 && shared[0] == 1 && shared[3] == 255);
    let ref u32[] returned = tail_u32(ref values);
    assert(returned.len == 3 && returned[0] == 127);
    let Option<ref u32[]> shared_result = shared_tagged_u32(ref values);
    switch (shared_result) { case .some(view): { assert(view.len == 3 && view[0] == 127); } case .none: { assert(false); } };
    let ref mut u32[] writable = vec::view_mut_u32(ref mut values);
    let ref mut u32[] child = writable[1..3];
    reverse_u32(child);
    read_u32(child);
    assert(writable[1] == 128 && writable[2] == 127);
    let ref mut u32[] nested = writable[1..][..1];
    nested[0] += 1;
    assert(writable[1] == 129);
    let Option<ref mut u32[]> result = tagged_u32(ref mut values);
    switch (result) { case .some(view): { view[0] = 130; } case .none: { assert(false); } };
    let ref mut u32[] loop_view = vec::view_mut_u32(ref mut values);
    var usize index = 0;
    for (let u32 byte in loop_view) {
        if (index == 0) { loop_view[1] = 42; }
        if (index == 1) { assert(byte == 42); }
        index += 1;
    }
    assert(index == 4);
    let usize capacity = vec::capacity<u32>(ref values);
    assert(switch (vec::reserve<u32>(ref mut values, capacity + 16)) {
        case .ok(new_capacity): { new_capacity >= capacity + 16 }
        case .err(_): { false }
    });
    let ref mut u32[] grown = vec::view_mut_u32(ref mut values);
    grown[2] = 99;
    assert(grown[1] == 42 && grown[2] == 99 && grown[3] == 255);
    append_u32(ref mut values, 7);
    assert(switch (vec::pop<u32>(ref mut values)) { case .some(byte): { byte == 7 } case .none: { false } });
    assert(vec::truncate<u32>(ref mut values, 2) == 2);
    assert(vec::clear<u32>(ref mut values) == 2);
    assert(vec::view_u32(ref values).is_empty);
    values = vec::empty<u32>();
    append_u32(ref mut values, 55);
    let ref mut u32[] last = vec::view_mut_u32(ref mut values);
    last[0] = 56;
    let vec::Vec<u32> moved = values;
    assert(vec::view_u32(ref moved)[0] == 56);
    var vec::Vec<u32> extremes = vec::empty<u32>();
    append_u32(ref mut extremes, 4294967295);
    assert(vec::view_u32(ref extremes)[0] == 4294967295);

}

fn void append_usize(ref mut vec::Vec<usize> values, usize value) {
    assert(switch (vec::push<usize>(values, value)) { case .ok(_): { true } case .err(_): { false } });
}
fn ref usize[] tail_usize(ref vec::Vec<usize> values) { return vec::view_usize(values)[1..]; }
fn Option<ref usize[]> shared_tagged_usize(ref vec::Vec<usize> values) { return .some(tail_usize(values)); }
fn ref mut usize[] middle_usize(ref mut vec::Vec<usize> values) { return vec::view_mut_usize(values)[1..3]; }
fn Option<ref mut usize[]> tagged_usize(ref mut vec::Vec<usize> values) { return .some(middle_usize(values)); }
fn void read_usize(ref usize[] values) { assert(values.len > 0); }
fn void reverse_usize(ref mut usize[] values) {
    var usize left = 0;
    var usize right = values.len;
    while (left < right) {
        right -= 1;
        let usize saved = values[left];
        values[left] = values[right];
        values[right] = saved;
        left += 1;
    }
}
fn void check_usize() {
    var usize[3] array = [1,2,3];
    let ref usize[] fixed_shared = ref array;
    var usize fixed_sum = 0;
    for (let usize item in fixed_shared) { fixed_sum += item; }
    assert(fixed_sum == 6);
    let ref mut usize[] fixed_mut = ref mut array;
    var usize fixed_index = 0;
    for (let usize item in fixed_mut) {
        if (fixed_index == 0) { fixed_mut[1] = 8; }
        if (fixed_index == 1) { assert(item == 8); }
        fixed_index += 1;
    }
    assert(array[1] == 8 && fixed_index == 3);
    var vec::Vec<usize> values = vec::empty<usize>();
    let ref usize[] empty = vec::view_usize(ref values);
    assert(empty.is_empty && empty[..].len == 0 && empty[0..0].is_empty);
    for (let usize byte in empty) { assert(false); }
    let ref mut usize[] writable_empty = vec::view_mut_usize(ref mut values);
    assert(writable_empty.is_empty && writable_empty[..].len == 0);
    for (let usize byte in writable_empty) { assert(false); }
    append_usize(ref mut values, 0);
    let ref mut usize[] one = vec::view_mut_usize(ref mut values);
    for (let usize byte in one) { assert(byte == 0); one[0] = 1; }
    append_usize(ref mut values, 127);
    append_usize(ref mut values, 128);
    append_usize(ref mut values, 255);
    let ref usize[] shared = vec::view_usize(ref values);
    var usize sum = 0;
    for (let usize byte in shared) { sum += byte; }
    assert(sum == 511 && shared[0] == 1 && shared[3] == 255);
    let ref usize[] returned = tail_usize(ref values);
    assert(returned.len == 3 && returned[0] == 127);
    let Option<ref usize[]> shared_result = shared_tagged_usize(ref values);
    switch (shared_result) { case .some(view): { assert(view.len == 3 && view[0] == 127); } case .none: { assert(false); } };
    let ref mut usize[] writable = vec::view_mut_usize(ref mut values);
    let ref mut usize[] child = writable[1..3];
    reverse_usize(child);
    read_usize(child);
    assert(writable[1] == 128 && writable[2] == 127);
    let ref mut usize[] nested = writable[1..][..1];
    nested[0] += 1;
    assert(writable[1] == 129);
    let Option<ref mut usize[]> result = tagged_usize(ref mut values);
    switch (result) { case .some(view): { view[0] = 130; } case .none: { assert(false); } };
    let ref mut usize[] loop_view = vec::view_mut_usize(ref mut values);
    var usize index = 0;
    for (let usize byte in loop_view) {
        if (index == 0) { loop_view[1] = 42; }
        if (index == 1) { assert(byte == 42); }
        index += 1;
    }
    assert(index == 4);
    let usize capacity = vec::capacity<usize>(ref values);
    assert(switch (vec::reserve<usize>(ref mut values, capacity + 16)) {
        case .ok(new_capacity): { new_capacity >= capacity + 16 }
        case .err(_): { false }
    });
    let ref mut usize[] grown = vec::view_mut_usize(ref mut values);
    grown[2] = 99;
    assert(grown[1] == 42 && grown[2] == 99 && grown[3] == 255);
    append_usize(ref mut values, 7);
    assert(switch (vec::pop<usize>(ref mut values)) { case .some(byte): { byte == 7 } case .none: { false } });
    assert(vec::truncate<usize>(ref mut values, 2) == 2);
    assert(vec::clear<usize>(ref mut values) == 2);
    assert(vec::view_usize(ref values).is_empty);
    values = vec::empty<usize>();
    append_usize(ref mut values, 55);
    let ref mut usize[] last = vec::view_mut_usize(ref mut values);
    last[0] = 56;
    let vec::Vec<usize> moved = values;
    assert(vec::view_usize(ref moved)[0] == 56);
    var vec::Vec<usize> extremes = vec::empty<usize>();
    append_usize(ref mut extremes, 18446744073709551615);
    assert(vec::view_usize(ref extremes)[0] == 18446744073709551615);

}

fn i32 main() { check_i32(); check_u32(); check_usize(); return 0; }
