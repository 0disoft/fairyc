module tests::large_arrays;
import array_types as arrays;
import reference_types as refs;
import tests::hir_nominal as support;
import mir_lower as lower;
import mir_model as mir;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;

@test fn void bounded_ids_and_large_array_owned_transport(){
    assert(arrays::make(1,64)==1537 && arrays::make(3,64)==1539);
    let u32 big=arrays::make(3,4096);assert(arrays::flat(big) && arrays::length(big)==4096 && arrays::element(big)==3);
    assert(refs::valid(refs::make(big,true)) && refs::byte_array(big));
    assert(!arrays::valid(33554952) && !arrays::valid(33587207));
    assert(arrays::make(1,0)==0 && arrays::make(1,4097)==0);
    assert(arrays::count(b"4_096")==4096 && arrays::count(b"0x1000")==4096 && arrays::count(b"4097")==0 && arrays::count(b"999999999999999999999999999999999999")==0);
    let p=support::retained(b"module main;struct Token{i32 n;}struct State{u8[4096] bytes;usize[1153] slots;Token[128] tokens;}fn i32 main(){var State s=State{.bytes=[0;4096],.slots=[0;1153],.tokens=[Token{.n=7};128]};s.bytes[4095]=9;s.slots[1152]=5;assert(s.bytes.len==4096 && s.slots.len==1153 && s.tokens.len==128);let State copy=s;assert(copy.bytes[4095]==9 && copy.slots[1152]==5);return copy.tokens[127].n;}");
    let m=switch(lower::lower(p)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    assert(verify::check(ref m).code==0);
    var bytes::Buffer out=bytes::empty();assert(c::render(ref m,ref mut out)==0);
    var bool found=false;var usize i=1;while(i<=mir::local_count(ref m)){if(mir::local_at(ref m,i).type==big){found=true;}i+=1;}assert(found);
}
