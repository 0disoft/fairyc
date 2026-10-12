module tests::foreign_calls;
import tests::hir_nominal as fixture;
import hir_model as h;
import mir_model as m;
import mir_lower as lower;
import mir_verify as verify;
import c_emit as c;
import pointer_types as pointers;
import c_abi_types as c_types;
import scalar_casts as casts;
import reference_types as refs;
import tagged_types as tags;
import array_types as arrays;
import std::bytes as bytes;
import std::vec as v;

fn m::Program changed(ref m::Program p,u32 mutation){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,m::source_view(p)).is_ok);
    var bytes::Buffer payload=bytes::empty();var v::Vec<m::Operation> ops=v::empty<m::Operation>();var usize i=1;
    while(i<=m::operation_count(p)){var o=m::operation_at(p,i);if(o.kind==38){if(mutation==9){o.boolean=false;}else if(mutation==10){o.owner_type=1048582;o.left=o.right;}else if(mutation==11){o.argument_count=0;}}if(o.kind==37){if(mutation==7){o.boolean=false;}else if(mutation==8){o.operator=1;}}if(o.kind==36){if(mutation==5){o.boolean=true;}else if(mutation==6){o.left=o.result;}}if(o.payload_end>o.payload_start){let data=m::literal_text(p,i);if(mutation==12 && o.kind==14 && o.type==12){assert(bytes::append(ref mut payload,data[..data.len-1]).is_ok && bytes::append_byte(ref mut payload,1).is_ok);}else{assert(bytes::append(ref mut payload,data).is_ok);}}assert(v::push<m::Operation>(ref mut ops,o).is_ok);i+=1;}
    var v::Vec<m::Target> targets=v::empty<m::Target>();i=1;while(i<=m::target_count(p)){var t=m::target_at(p,i);if(t.foreign){if(mutation==1){t.link_start=7;t.link_end=11;}else if(mutation==2){t.result=6;}else if(mutation==3){t.foreign=false;}else if(mutation==4){t.unsafe_call=false;}}assert(v::push<m::Target>(ref mut targets,t).is_ok);i+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();i=0;while(i<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,i)}).is_ok);i+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();i=1;while(i<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();i=1;while(i<=m::local_count(p)){assert(v::push<m::Local>(ref mut locals,m::local_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();i=1;while(i<=m::block_count(p)){var b=m::block_at(p,i);assert(v::push<m::Block>(ref mut blocks,b).is_ok);i+=1;}
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<m::edge_count(p)){assert(v::push<usize>(ref mut edges,m::edge_at(p,i)).is_ok);i+=1;}
    return switch(m::assemble(source,payload,targets,signatures,functions,locals,blocks,ops,edges,0)){case .ok(p):{p}case .err(_):{assert(false);return changed(p,mutation);}};
}
@test fn void foreign_targets_reject_mutated_linker_identity(){
    let p=switch(lower::lower(fixture::retained(b"module main;@link_name(\"fixture_add\")unsafe extern C fn i32 foreign(i32 n);fn i32 main(){return unsafe{foreign(6)};}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0 && m::function_count(ref p)==1 && m::target_count(ref p)==2);
    let t=m::target_at(ref p,1);assert(t.foreign && t.external && t.unsafe_call && bytes::equal(m::foreign_text(ref p,1),b"fixture_add"));
    var bytes::Buffer original=bytes::empty();assert(c::render(ref p,ref mut original)==0);
    var u32 mutation=1;while(mutation<=4){let bad=changed(ref p,mutation);assert(verify::check(ref bad).code==if(mutation==3){1001}else{5106});
        var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;}
}

@test fn void raw_null_operations_reject_forged_operands(){
    let p=switch(lower::lower(fixture::retained(b"module main;unsafe extern C fn i32 fixture_null(ptr u8 p);fn i32 main(){return unsafe{fixture_null(null)};}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);
    var u32 mutation=5;while(mutation<=6){let bad=changed(ref p,mutation);assert(verify::check(ref bad).code!=0);var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;}
}

@test fn void pointer_type_ids_are_disjoint_from_tags_arrays_and_borrows(){
    var u32 element=1;while(element<=7){if(element!=6){let u32 shared=pointers::make(element,false);let u32 exclusive=pointers::make(element,true);assert(pointers::valid(shared) && pointers::valid(exclusive));assert(pointers::inner(shared)==element && !pointers::mutable(shared) && pointers::mutable(exclusive));assert(!tags::builtin(shared) && !arrays::valid(shared));}element+=1;}
    assert(pointers::make(6,false)==0 && !pointers::valid(196610) && !pointers::valid(1048588));
}

@test fn void pointer_casts_reject_lost_provenance(){
    let p=switch(lower::lower(fixture::retained(b"module main;fn i32 main(){let i32 n=7;let ptr i32 p=unsafe{ref n as ptr i32};assert(p!=null);return 7;}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);var u32 mutation=7;while(mutation<=8){let bad=changed(ref p,mutation);assert(verify::check(ref bad).code==1001);var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;}
}

@test fn void raw_views_reject_lost_owner_or_unsafe_permission(){
    let p=switch(lower::lower(fixture::retained(b"module main;fn i32 main(){let ref u8[] owner=b\"AB\";let ptr u8 data=unsafe{owner as ptr u8};let ref u8[] v=unsafe{byte_slice::from_raw_parts(owner,data,2)};assert(v[0]==65);return 7;}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);
    var u32 mutation=9;while(mutation<=11){let bad=changed(ref p,mutation);assert(verify::check(ref bad).code!=0);
        var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);
        assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;}
}

@test fn void c_scalar_identity_and_cast_lane_are_closed(){
    var u32 type=128;while(type<=143){assert(c_types::valid(type));assert(!pointers::valid(type) && !tags::builtin(type) && !arrays::valid(type));type+=1;}
    assert(c_types::named(b"c::int")==132 && c_types::named(b"c::longdouble")==0 && c_types::named(b"c::ssize_t")==138);
    assert(casts::allowed(1,132) && casts::allowed(132,1));assert(!casts::allowed(4,140) && !casts::allowed(128,6));
    let p=switch(lower::lower(fixture::retained(b"module main;@link_name(\"fixture_make_int\")unsafe extern C fn c::int host();fn i32 main(){let c::int n=unsafe{host()};return n as i32;}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);let bad=changed(ref p,2);assert(verify::check(ref bad).code!=0);
    var bytes::Buffer out=bytes::empty();assert(c::render(ref p,ref mut out)==0);
}

@test fn void only_mutable_opaque_pointer_output_slots_are_admitted(){
    let u32 pointer=pointers::make(7,true);let u32 slot=pointers::make(pointer,true);let u32 borrowed=refs::make(pointer,true);
    assert(pointer==1048591 && slot==1048601 && pointers::valid(slot) && pointers::inner(slot)==pointer && pointers::mutable(slot));
    assert(refs::valid(borrowed) && refs::inner(borrowed)==pointer && refs::mutable(borrowed) && pointers::cast_allowed(borrowed,slot));
    assert(pointers::make(pointer,false)==0 && pointers::make(pointers::make(3,true),true)==0);
    assert(!pointers::cast_allowed(refs::make(pointer,false),slot) && tags::make_option(borrowed)==0);
}

@test fn void cstr_storage_and_raw_origin_are_independently_verified(){
    let p=switch(lower::lower(fixture::retained(b"module main;fn i32 main(){let ref CStr s=c\"AB\";let ptr u8 p=unsafe{s as ptr u8};assert(p!=null);return 7;}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);let bad=changed(ref p,12);assert(verify::check(ref bad).code!=0);var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));
    let raw=switch(lower::lower(fixture::retained(b"module main;fn i32 main(){let ref CStr owner=c\"AB\";let ptr u8 data=unsafe{owner as ptr u8};let ref CStr s=unsafe{CStr::from_raw_parts(owner,data,3)};let ptr u8 p=unsafe{s as ptr u8};assert(p!=null);return 7;}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref raw).code==0);var u32 mutation=9;while(mutation<=11){let invalid=changed(ref raw,mutation);assert(verify::check(ref invalid).code!=0);mutation+=1;}
}
