module main;

fn i64 sum(ref i64[] view) {
    var i64 total = 0;
    for (let i64 value in view) { total += value; }
    return total;
}

fn i64 selected_sum(ref i64[] view) {
    var i64 total = 0;
    for (let i64 value in view) {
        if (value < 0) { continue; }
        if (value == 42) { break; }
        total += value;
    }
    return total;
}

fn i64 find_extreme(ref i64[] view) {
    for (let i64 value in view) {
        if (value == -9223372036854775808) { return value; }
        if (value == 9223372036854775807) { return value; }
    }
    return 0;
}

fn i32 nested(ref i64[] outer, ref i64[] inner) {
    var i32 count = 0;
    for (let i64 value in outer) {
        for (let i64 value in inner) {
            if (value < 0) { continue; }
            count += 1;
            if (count == 3) { return count; }
            break;
        }
    }
    return count;
}

fn i32 main() {
    var i64[5] data = [4000000000, -3, 7, 42, 99];
    let ref i64[] view = ref data;
    assert(sum(view) == 4000000145);
    assert(selected_sum(view) == 4000000007);
    let ref i64[] middle = view[1..3];
    assert(sum(middle) == 4);
    let ref i64[] empty = view[5..];
    var usize visits = 0;
    for (let i64 value in empty) { visits += 1; }
    assert(visits == 0 && sum(empty) == 0);
    let ref i64[] singleton = view[2..3];
    assert(sum(singleton) == 7 && nested(view, middle) == 3);
    data[0] = 12;
    let ref i64[] changed = ref data;
    assert(sum(changed) == 157);

    let i64[2] extremes = [-9223372036854775808, 9223372036854775807];
    let ref i64[] extreme_view = ref extremes;
    assert(sum(extreme_view) == -1);
    assert(find_extreme(extreme_view) == -9223372036854775808);
    let ref i64[] maximum = extreme_view[1..];
    assert(find_extreme(maximum) == 9223372036854775807);
    return 0;
}
