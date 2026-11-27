module main;

unsafe extern C fn i32 legacy_filter(
    unsafe extern C fn i32(i32, ptr mut void) predicate, ptr mut void data);
unsafe extern C fn i32 legacy_note_visit(ptr mut void data);
pub unsafe extern C fn i32 legacy_report(i32 accepted, i32 visits);

// A normal Fairy function becomes a synchronous C callback at this boundary.
pub unsafe fn i32 keep_large(i32 value, ptr mut void data) {
    let i32 visits = legacy_note_visit(data);
    assert(visits > 0);
    return if(value >= 10){1}else{0};
}

fn i32 main() {
    var i32 visits = 0;
    let i32 accepted = unsafe {
        legacy_filter(keep_large, ref mut visits as ptr mut void)
    };
    assert(accepted == 2);
    assert(visits == 5);
    return unsafe { legacy_report(accepted, visits) };
}
