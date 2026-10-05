module mir_verify;
import mir_model as m;
import mir_verify_shape as shape;
import mir_verify_flow as flow;

// Scalar invariants V1001, V1002 and V1501. Ownership/borrow/Region/unsafe
// invariants remain closed until the corresponding value forms are admitted.
pub fn m::Problem check(ref m::Program p) {
    let m::Problem structural = shape::check(p);
    if (structural.code != 0) { return structural; }
    var flow::Budget budget = flow::Budget { .work = 0 }; var usize fid = 1;
    while (fid <= m::function_count(p)) {
        let m::Problem result = flow::function(p,fid,ref mut budget);
        if (result.code != 0) { return result; } fid += 1;
    }
    return shape::problem(0,0,0);
}
