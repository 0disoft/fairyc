module mir_verify;
import mir_model as m;
import mir_verify_shape as shape;
import mir_verify_flow as flow;
import mir_verify_arrays as arrays;
import mir_verify_literals as literals;
import mir_verify_views as views;

// Scalar invariants V1001, V1002 and V1501. Ownership/borrow/Region/unsafe
// invariants remain closed until the corresponding value forms are admitted.
pub fn m::Problem check(ref m::Program p) {
    let m::Problem structural = shape::check(p);
    if (structural.code != 0) { return structural; }
    let m::Problem indexes = arrays::check(p); if (indexes.code != 0) { return indexes; }
    let m::Problem views = literals::check(p); if (views.code != 0) { return views; }
    var flow::Budget budget = flow::Budget { .work = 0 }; var usize fid = 1;
    while (fid <= m::function_count(p)) {
        let m::Problem result = flow::function(p,fid,ref mut budget);
        if (result.code != 0) { return result; } fid += 1;
    }
    return views::check(p);
}
