module tests::must_use;
import tests::hir_nominal as fixture;
import hir_model as h;
import must_use_flow as uses;
@test fn void declaration_marker_is_owned_and_inert_only(){
    let p=fixture::retained(b"module main;@must_use fn i32 token(){return 7;}fn i32 main(){return token();}");
    assert(h::binding_at(ref p,1).must_use && !h::binding_at(ref p,2).must_use);
    assert(uses::eligible(1) && uses::eligible(2) && uses::eligible(6));
    assert(!uses::eligible(0) && !uses::eligible(7) && !uses::eligible(9) && !uses::eligible(10) && !uses::eligible(11) && !uses::eligible(98304));
}
