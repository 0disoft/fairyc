module tests::project_diagnostic_codes;
import project_diagnostic_codes as diagnostics;

@test
fn void overlapping_internal_statuses_do_not_become_resolution_diagnostics(){
    assert(diagnostics::code(1,0,2)==102);
    assert(diagnostics::code(3,2,5)==101);
    assert(diagnostics::code(2,0,5)==0);
    assert(diagnostics::code(3,4,2)==0);
    assert(diagnostics::code(3,7,10)==0);
    assert(diagnostics::code(3,6,1002)==0);
    assert(diagnostics::code(4,0,1010)==0);
    assert(diagnostics::code(0,0,4202)==0);
}
