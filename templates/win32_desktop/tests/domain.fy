module tests::domain;
import domain as domain;
// Buffer-based transformation is exercised by verify_cli.py on the native EXE.
@test fn void navigation_policy(){
    assert(domain::key_action(9,0)==2);assert(domain::key_action(9,1)==3);
    assert(domain::key_action(13,0)==1);assert(domain::key_action(27,0)==4);assert(domain::key_action(13,32)==0);
}
