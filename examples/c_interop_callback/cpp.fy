module cpp;
import main as callback;

unsafe extern C fn i32 legacy_cpp_filter(
    i32 force_exception, ptr mut i32 releases,
    unsafe extern C fn i32(i32, ptr mut void) predicate, ptr mut void data);

fn i32 run_case(i32 force_exception, i32 expected) {
    var i32 visits = 0;
    var i32 releases = 0;
    let i32 status = unsafe {
        legacy_cpp_filter(force_exception, ref mut releases as ptr mut i32,
            callback::keep_large, ref mut visits as ptr mut void)
    };
    assert(status == expected);
    assert(visits == 5);
    assert(releases == 1);
    return visits;
}

fn i32 main() {
    let i32 visits = run_case(0, 2);
    discard run_case(1, -2);
    return unsafe { callback::legacy_report(2, visits) };
}
