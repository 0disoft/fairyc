module tests::project_diagnostic_codes;
import project_diagnostic_codes as diagnostics;
import std::bytes as bytes;

@test
fn void overlapping_internal_statuses_do_not_become_resolution_diagnostics(){
    assert(diagnostics::code(1,0,2)==102);
    assert(diagnostics::code(3,2,5)==101);
    assert(diagnostics::code(3,3,1)==101);
    assert(diagnostics::code(2,0,1)==0);
    assert(diagnostics::code(3,4,1)==0);
    assert(diagnostics::code(3,4,103)==103);
    assert(diagnostics::code(2,0,5)==0);
    assert(diagnostics::code(3,4,2)==0);
    assert(diagnostics::code(3,7,10)==0);
    assert(diagnostics::code(3,6,1002)==0);
    assert(diagnostics::code(4,0,1010)==0);
    assert(diagnostics::code(0,0,4202)==0);
}

@test
fn void argument_label_failures_keep_their_public_code_in_analysis_phases(){
    assert(diagnostics::code(3,4,1014)==1014 && diagnostics::code(3,6,1014)==1014);
    assert(bytes::equal(diagnostics::summary(1014),b"call argument label does not match its position or target"));
    assert(diagnostics::code(4,0,1014)==0 && diagnostics::code(3,6,14)==0);
}
